#!/usr/bin/env node

/**
 * XRPL Faucet Request Module
 *
 * This module handles requesting funds from XRPL faucets.
 *
 * Usage: node faucet.js <config-json-path>
 *
 * Config JSON format:
 * {
 *   "faucet_url": "https://faucet.altnet.rippletest.net/accounts",
 *   "wallet_seed": "sXXX..." (optional),
 *   "wallet_address": "rXXX..." (optional),
 *   "network_url": "wss://..." (optional, for balance check),
 *   "is_local": false (optional, if true uses genesis account for funding),
 *   "verbose": true (optional)
 * }
 *
 * Output JSON format:
 * {
 *   "success": true,
 *   "data": {
 *     "txHash": "...",
 *     "walletAddress": "...",
 *     "walletSeed": "...",
 *     "balance": "1000",
 *     "faucetAmount": "1000"
 *   }
 * }
 */

const xrpl = require('@xrpl-commons/xrpl');
const fs = require('fs');
const https = require('https');
const http = require('http');

// Genesis account seed for local development
const GENESIS_SEED = 'snoPBrXtMeMyMHUVTgbuqAfg1SUTb';
// Default amount to fund in drops (1000 XRP)
const LOCAL_FAUCET_AMOUNT = '1000000000';

/**
 * Request funds from XRPL faucet
 */
async function requestFaucet(config) {
  const { faucet_url, wallet_seed, wallet_address, network_url, is_local, verbose } =
    config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Requesting funds from faucet...\n');

  try {
    let wallet;
    let address;

    // Determine wallet/address
    const algorithm = config.algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    if (wallet_seed) {
      wallet = algorithm ? xrpl.Wallet.fromSeed(wallet_seed, { algorithm }) : xrpl.Wallet.fromSeed(wallet_seed);
      address = wallet.address;
      log('Using provided wallet seed');
      log('  Address:', address);
    } else if (wallet_address) {
      address = wallet_address;
      log('Using provided address:', address);
    } else {
      wallet = algorithm ? xrpl.Wallet.generate(algorithm) : xrpl.Wallet.generate();
      address = wallet.address;
      log('Generated new wallet');
      log('  Address:', address);
      log('  Seed:', wallet.seed);
    }

    let faucetResult;

    // Use local genesis funding or external faucet
    if (is_local) {
      log('\nFunding from local genesis account...');
      log('  Network URL:', network_url);
      faucetResult = await fundFromGenesis(network_url, address, log);
      log('✓ Local funding successful');
    } else {
      log('\nRequesting funds from faucet...');
      log('  Faucet URL:', faucet_url);
      faucetResult = await makeFaucetRequest(faucet_url, address);
      log('✓ Faucet request successful');
    }

    // Get balance if network URL provided
    let balance = null;
    if (network_url) {
      log('\nChecking balance...');
      const client = new xrpl.Client(network_url);
      client.apiVersion = 1;
      await client.connect();
      balance = await client.getXrpBalance(address);
      await client.disconnect();
      log('  Balance:', balance, 'XRP');
    }

    // Output result
    const result = {
      success: true,
      data: {
        txHash: faucetResult.txHash,
        walletAddress: address,
        walletSeed: wallet ? wallet.seed : '',
        balance: balance ? String(balance) : '',
        faucetAmount: faucetResult.amount ? String(faucetResult.amount) : '',
      },
    };

    console.log(JSON.stringify(result));
    return result;
  } catch (error) {
    const errorResult = {
      success: false,
      error: error.message,
      details: error.stack,
    };

    console.log(JSON.stringify(errorResult));
    process.exit(1);
  }
}

/**
 * Fund an address from the local genesis account
 */
async function fundFromGenesis(networkUrl, destinationAddress, log) {
  const client = new xrpl.Client(networkUrl);
  client.apiVersion = 1;
  await client.connect();

  try {
    // Create genesis wallet
    const genesisWallet = xrpl.Wallet.fromSeed(GENESIS_SEED, { algorithm: xrpl.ECDSA.secp256k1 });
    log('  Genesis address:', genesisWallet.address);

    // Prepare Payment transaction
    const payment = {
      TransactionType: 'Payment',
      Account: genesisWallet.address,
      Destination: destinationAddress,
      Amount: LOCAL_FAUCET_AMOUNT,
    };

    // Autofill, sign, and submit
    const prepared = await client.autofill(payment);
    const signed = genesisWallet.sign(prepared);
    const result = await client.submitAndWait(signed.tx_blob);

    if (result.result.meta.TransactionResult !== 'tesSUCCESS') {
      throw new Error(`Transaction failed: ${result.result.meta.TransactionResult}`);
    }

    return {
      txHash: result.result.hash,
      amount: String(Number(LOCAL_FAUCET_AMOUNT) / 1000000), // Convert drops to XRP
    };
  } finally {
    await client.disconnect();
  }
}

/**
 * Make HTTP request to faucet
 */
function makeFaucetRequest(faucetUrl, address) {
  return new Promise((resolve, reject) => {
    const url = new URL(faucetUrl);
    const protocol = url.protocol === 'https:' ? https : http;

    const postData = JSON.stringify({
      destination: address,
    });

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

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        if (res.statusCode === 200 || res.statusCode === 201) {
          try {
            const response = JSON.parse(data);
            // Different faucets return different formats
            // Try to extract relevant info
            const txHash =
              response.hash || response.txHash || response.tx_hash || 'unknown';
            const amount = response.amount || response.balance || 'unknown';

            resolve({
              txHash: String(txHash),
              amount: String(amount),
            });
          } catch (err) {
            reject(
              new Error(`Failed to parse faucet response: ${err.message}`)
            );
          }
        } else {
          reject(
            new Error(
              `Faucet request failed with status ${res.statusCode}: ${data}`
            )
          );
        }
      });
    });

    req.on('error', (err) => {
      reject(new Error(`Faucet request failed: ${err.message}`));
    });

    req.write(postData);
    req.end();
  });
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error(`
Usage: node faucet.js <config-json-path>

The config JSON file should contain:
{
  "faucet_url": "https://faucet.altnet.rippletest.net/accounts",
  "wallet_seed": "sXXX..." (optional),
  "wallet_address": "rXXX..." (optional),
  "network_url": "wss://..." (optional),
  "verbose": true (optional)
}

Output is pure JSON to stdout.
`);
    process.exit(1);
  }

  const configPath = args[0];

  try {
    let configContent;
    if (!configPath || configPath === '-') {
      // Config piped via stdin; reading fd 0 synchronously is supported in Node 12+.
      configContent = fs.readFileSync(0, 'utf8');
    } else if (fs.existsSync(configPath)) {
      configContent = fs.readFileSync(configPath, 'utf8');
    } else {
      const errorResult = {
        success: false,
        error: `Config file not found: ${configPath}`,
        details: 'Provide config via stdin (recommended) or as a valid file path',
      };
      console.log(JSON.stringify(errorResult));
      process.exit(1);
    }
    const config = JSON.parse(configContent);
    requestFaucet(config);
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

module.exports = { requestFaucet };
