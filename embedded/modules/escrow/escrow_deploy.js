#!/usr/bin/env node

/**
 * XRPL Smart Escrow Deployment Module
 *
 * This module handles deploying smart escrows with WASM finish conditions to XRPL networks.
 *
 * Usage: node escrow_deploy.js <config-json-path>
 *
 * Config JSON format:
 * {
 *   "wasm_path": "/path/to/contract.wasm",
 *   "destination": "rXXX...",
 *   "amount": "1000000" (in drops),
 *   "cancel_after": 123456789 (optional, ripple epoch),
 *   "finish_after": 123456789 (optional, ripple epoch),
 *   "network_url": "wss://alphanet.xrpl.org",
 *   "network_id": 21465,
 *   "wallet_seed": "sXXX..." (optional),
 *   "faucet_url": "https://faucet..." (optional),
 *   "fee": "1000000" (optional, default 1 XRP),
 *   "verbose": true (optional),
 *   "algorithm": "secp256k1" (optional)
 * }
 *
 * Output JSON format:
 * {
 *   "success": true,
 *   "data": {
 *     "txHash": "...",
 *     "walletAddress": "...",
 *     "walletSeed": "...",
 *     "escrowSequence": 123,
 *     "validated": true
 *   }
 * }
 */

const xrpl = require('@xrpl-commons/xrpl');
const fs = require('fs');
const http = require('http');
const https = require('https');

/**
 * Send a JSON-RPC request via HTTP
 */
function httpRPC(rpcUrl, method, params, timeout) {
  return new Promise((resolve, reject) => {
    const url = new URL(rpcUrl);
    const protocol = url.protocol === 'https:' ? https : http;
    const postData = JSON.stringify({ method, params: [params] });
    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData),
      },
    };
    const req = protocol.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          if (parsed.result && parsed.result.status === 'error') {
            reject(new Error(`${parsed.result.error}: ${parsed.result.error_message}`));
          } else {
            resolve(parsed.result);
          }
        } catch (e) {
          reject(new Error(`Failed to parse RPC response: ${e.message}`));
        }
      });
    });
    req.on('error', (err) => reject(err));
    req.setTimeout(timeout || 30000, () => { req.destroy(); reject(new Error('RPC request timeout')); });
    req.write(postData);
    req.end();
  });
}

/**
 * Deploy a smart escrow to XRPL network
 */
async function deployEscrow(config) {
  const {
    wasm_path,
    destination,
    amount,
    cancel_after,
    finish_after,
    network_url,
    network_id,
    wallet_seed,
    faucet_url,
    fee,
    verbose,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Deploying smart escrow to XRPL...\n');

  const client = new xrpl.Client(network_url);
  client.apiVersion = 1;

  try {
    await client.connect();
    log('Connected to network');

    // Create or restore wallet
    const algorithm = config.algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    const wallet = wallet_seed
      ? (algorithm ? xrpl.Wallet.fromSeed(wallet_seed, { algorithm }) : xrpl.Wallet.fromSeed(wallet_seed))
      : (algorithm ? xrpl.Wallet.generate(algorithm) : xrpl.Wallet.generate());

    log('\nWallet:');
    log('  Address:', wallet.address);
    log('  Seed:', wallet.seed);

    // Read WASM file
    if (!fs.existsSync(wasm_path)) {
      throw new Error(`WASM file not found: ${wasm_path}`);
    }

    const wasmBytes = fs.readFileSync(wasm_path);
    const wasmHex = wasmBytes.toString('hex').toUpperCase();

    log(`\nWASM size: ${wasmBytes.length} bytes`);

    // Check balance and auto-fund if needed
    let balance = 0;
    try {
      balance = await client.getXrpBalance(wallet.address);
    } catch (e) {
      balance = 0;
    }
    log(`\nWallet balance: ${balance} XRP`);

    if (parseFloat(balance) === 0 && faucet_url) {
      log('Wallet not funded, requesting funds...');
      const isLocal = network_url.includes('localhost') || network_url.includes('127.0.0.1');
      if (isLocal) {
        const GENESIS_SEED = 'snoPBrXtMeMyMHUVTgbuqAfg1SUTb';
        const genesisWallet = xrpl.Wallet.fromSeed(GENESIS_SEED, { algorithm: xrpl.ECDSA.secp256k1 });
        const payment = {
          TransactionType: 'Payment',
          Account: genesisWallet.address,
          Destination: wallet.address,
          Amount: '1000000000',
        };
        const prepared = await client.autofill(payment);
        const signed = genesisWallet.sign(prepared);
        const fundResult = await client.submitAndWait(signed.tx_blob);
        if (fundResult.result.meta.TransactionResult !== 'tesSUCCESS') {
          throw new Error(`Funding failed: ${fundResult.result.meta.TransactionResult}`);
        }
        log('Funded from local genesis account');
      } else {
        await new Promise((resolve, reject) => {
          const url = new URL(faucet_url);
          const protocol = url.protocol === 'https:' ? https : http;
          const postData = JSON.stringify({ destination: wallet.address });
          const options = {
            hostname: url.hostname,
            port: url.port,
            path: url.pathname + url.search,
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Content-Length': Buffer.byteLength(postData),
            },
          };
          const req = protocol.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => { data += chunk; });
            res.on('end', () => {
              if (res.statusCode === 200 || res.statusCode === 201) {
                resolve(data);
              } else {
                reject(new Error(`Faucet request failed: ${res.statusCode} ${data}`));
              }
            });
          });
          req.on('error', (err) => reject(err));
          req.write(postData);
          req.end();
        });
        log('Funded from external faucet');
      }
      await new Promise(r => setTimeout(r, 2000));
      balance = await client.getXrpBalance(wallet.address);
      log(`Updated balance: ${balance} XRP`);
    } else if (parseFloat(balance) === 0) {
      log('Warning: Wallet not funded and no faucet URL provided, deployment will likely fail');
    }

    // Build EscrowCreate transaction
    log('\nSubmitting EscrowCreate transaction...');

    const accountInfo = await client.request({
      command: 'account_info',
      account: wallet.address,
    });

    const tx = {
      TransactionType: 'EscrowCreate',
      Account: wallet.address,
      Destination: destination,
      Amount: amount,
      FinishFunction: wasmHex,
      Fee: fee || '1000000',
      Sequence: accountInfo.result.account_data.Sequence,
      SigningPubKey: wallet.publicKey,
    };

    if (network_id && network_id > 1024) {
      tx.NetworkID = network_id;
    }

    if (cancel_after !== undefined && cancel_after !== null) {
      tx.CancelAfter = cancel_after;
    }

    if (finish_after !== undefined && finish_after !== null) {
      tx.FinishAfter = finish_after;
    }

    const escrowSequence = tx.Sequence;
    const signed = wallet.sign(tx);

    log('Transaction ID:', signed.hash);

    const isLocal = network_url.includes('localhost') || network_url.includes('127.0.0.1');

    let txResult = null;

    if (isLocal) {
      // Local nodes: use HTTP RPC to avoid WebSocket hangs under emulation
      await client.disconnect();

      const rpcUrl = network_url
        .replace('ws://', 'http://').replace('wss://', 'https://')
        .replace('localhost:6006', 'localhost:5005');

      const submitResult = await httpRPC(rpcUrl, 'submit', { tx_blob: signed.tx_blob }, 120000);

      log('Submit response:', JSON.stringify(submitResult).substring(0, 200));

      if (submitResult.engine_result &&
          submitResult.engine_result !== 'tesSUCCESS' &&
          !submitResult.engine_result.startsWith('tes')) {
        throw new Error(`Transaction rejected: ${submitResult.engine_result} - ${submitResult.engine_result_message}`);
      }

      // Wait for validation by polling tx via HTTP RPC
      for (let attempt = 0; attempt < 60; attempt++) {
        await new Promise(r => setTimeout(r, 1000));
        try {
          const txRes = await httpRPC(rpcUrl, 'tx', { transaction: signed.hash }, 10000);
          if (txRes.validated) {
            txResult = txRes;
            break;
          }
        } catch (e) {
          // Transaction not yet found, keep waiting
        }
      }
    } else {
      // Remote networks: use WebSocket submit and wait
      const submitResult = await client.request({
        command: 'submit',
        tx_blob: signed.tx_blob,
      });

      log('Submit response:', JSON.stringify(submitResult.result).substring(0, 200));

      if (submitResult.result.engine_result &&
          submitResult.result.engine_result !== 'tesSUCCESS' &&
          !submitResult.result.engine_result.startsWith('tes')) {
        throw new Error(`Transaction rejected: ${submitResult.result.engine_result} - ${submitResult.result.engine_result_message}`);
      }

      // Poll for validation via WebSocket
      for (let attempt = 0; attempt < 60; attempt++) {
        await new Promise(r => setTimeout(r, 1000));
        try {
          const txRes = await client.request({
            command: 'tx',
            transaction: signed.hash,
          });
          if (txRes.result.validated) {
            txResult = txRes.result;
            break;
          }
        } catch (e) {
          // Transaction not yet found, keep waiting
        }
      }

      await client.disconnect();
    }

    if (!txResult) {
      throw new Error('Transaction was not validated within 60 seconds');
    }

    log('\nSmart escrow created successfully!');
    log('Escrow Sequence:', escrowSequence);

    const result = {
      success: true,
      data: {
        txHash: signed.hash,
        walletAddress: wallet.address,
        walletSeed: wallet.seed,
        escrowSequence: escrowSequence,
        validated: txResult.validated,
      },
    };

    console.log(JSON.stringify(result));
    return result;
  } catch (error) {
    if (client.isConnected()) {
      await client.disconnect();
    }

    const errorResult = {
      success: false,
      error: error.message,
      details: error.data ? JSON.stringify(error.data) : error.stack,
    };

    console.log(JSON.stringify(errorResult));
    process.exit(1);
  }
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error(`
Usage: node escrow_deploy.js <config-json-path>

The config JSON file should contain:
{
  "wasm_path": "/path/to/contract.wasm",
  "destination": "rXXX...",
  "amount": "1000000",
  "cancel_after": 123456789 (optional),
  "finish_after": 123456789 (optional),
  "network_url": "wss://alphanet.xrpl.org",
  "network_id": 21465,
  "wallet_seed": "sXXX..." (optional),
  "faucet_url": "https://faucet..." (optional),
  "fee": "1000000" (optional, default 1 XRP),
  "verbose": true (optional),
  "algorithm": "secp256k1" (optional)
}

Output is pure JSON to stdout.
`);
    process.exit(1);
  }

  const configPath = args[0];

  if (!fs.existsSync(configPath)) {
    const errorResult = {
      success: false,
      error: `Config file not found: ${configPath}`,
      details: 'Please provide a valid config JSON file path',
    };
    console.log(JSON.stringify(errorResult));
    process.exit(1);
  }

  try {
    const configContent = fs.readFileSync(configPath, 'utf8');
    const config = JSON.parse(configContent);
    deployEscrow(config);
  } catch (error) {
    const errorResult = {
      success: false,
      error: 'Failed to load config',
      details: error.message,
    };
    console.log(JSON.stringify(errorResult));
    process.exit(1);
  }
}

module.exports = { deployEscrow };
