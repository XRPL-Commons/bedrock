#!/usr/bin/env node

/**
 * XRPL Smart Vault Withdraw Module
 *
 * This module handles withdrawing from a smart vault on XRPL networks via VaultWithdraw transactions.
 *
 * Usage: node vault_withdraw.js <config-json-path>
 *
 * Config JSON format:
 * {
 *   "vault_id": "AAAA...",
 *   "amount": "1000000",
 *   "destination": "rXXX...",
 *   "network_url": "wss://alphanet.xrpl.org",
 *   "network_id": 21465,
 *   "wallet_seed": "sXXX...",
 *   "computation_allowance": "1000000" (optional, default 1000000),
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
 * Withdraw from a smart vault on XRPL network
 */
async function vaultWithdraw(config) {
  const log = config.verbose ? console.error.bind(console) : () => {};

  log('Withdrawing from vault on XRPL...\n');

  const client = new xrpl.Client(config.network_url);
  client.apiVersion = 1;

  try {
    await client.connect();
    log('Connected to network');

    // Restore wallet
    const algorithm = config.algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    const wallet = algorithm
      ? xrpl.Wallet.fromSeed(config.wallet_seed, { algorithm })
      : xrpl.Wallet.fromSeed(config.wallet_seed);

    log('\nWallet:');
    log('  Address:', wallet.address);

    // Build VaultWithdraw transaction
    log('\nSubmitting VaultWithdraw transaction...');

    const accountInfo = await client.request({
      command: 'account_info',
      account: wallet.address,
    });

    const tx = {
      TransactionType: 'VaultWithdraw',
      Account: wallet.address,
      VaultID: config.vault_id,
      Amount: config.amount,
      Destination: config.destination,
      ComputationAllowance: parseInt(config.computation_allowance || '1000000'),
      Fee: config.fee || '1000000',
      Sequence: accountInfo.result.account_data.Sequence,
      SigningPubKey: wallet.publicKey,
    };

    if (config.network_id && config.network_id > 1024) {
      tx.NetworkID = config.network_id;
    }

    const signed = wallet.sign(tx);

    log('Transaction ID:', signed.hash);

    const isLocal = config.network_url.includes('localhost') || config.network_url.includes('127.0.0.1');

    let txResult = null;

    if (isLocal) {
      // Local nodes: use HTTP RPC to avoid WebSocket hangs under emulation
      await client.disconnect();

      const rpcUrl = config.network_url
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

    log('\nWithdrawal successful!');
    log('\nDone');

    const result = {
      success: true,
      data: {
        txHash: signed.hash,
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
Usage: node vault_withdraw.js <config-json-path>

The config JSON file should contain:
{
  "vault_id": "AAAA...",
  "amount": "1000000",
  "destination": "rXXX...",
  "network_url": "wss://alphanet.xrpl.org",
  "network_id": 21465,
  "wallet_seed": "sXXX...",
  "fee": "1000000" (optional),
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
    vaultWithdraw(config);
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

module.exports = { vaultWithdraw };
