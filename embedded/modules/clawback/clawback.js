#!/usr/bin/env node

/**
 * XRPL Smart Contract Clawback Module
 *
 * Handles clawing back tokens from contracts via ContractClawback transaction.
 * This allows token issuers to reclaim tokens held by a contract.
 *
 * Usage: node clawback.js <config-json-path>
 */

const xrpl = require('xrpl');
const fs = require('fs');

async function clawbackContract(config) {
  const {
    contract_account,
    amount,
    network_url,
    wallet_seed,
    algorithm,
    fee,
    verbose,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Clawing back tokens from contract...\n');

  const client = new xrpl.Client(network_url);
  client.apiVersion = 1;

  try {
    await client.connect();
    log('Connected to network');

    const algo = algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    const wallet = algo
      ? xrpl.Wallet.fromSeed(wallet_seed, { algorithm: algo })
      : xrpl.Wallet.fromSeed(wallet_seed);

    // Parse the amount - could be drops string or currency object
    let parsedAmount;
    if (typeof amount === 'string' && amount.includes('/')) {
      // Format: "value/currency/issuer"
      const parts = amount.split('/');
      if (parts.length !== 3) {
        throw new Error('Amount format must be "value/currency/issuer" or drops string');
      }
      parsedAmount = {
        value: parts[0],
        currency: parts[1],
        issuer: parts[2],
      };
    } else {
      // Assume it's already a parsed object or drops string
      parsedAmount = amount;
    }

    const tx = {
      TransactionType: 'ContractClawback',
      Account: wallet.address,
      ContractAccount: contract_account,
      Amount: parsedAmount,
      Fee: fee || '1000000',
    };

    log('Transaction:', JSON.stringify(tx, null, 2));

    const prepared = await client.autofill(tx);
    const signed = wallet.sign(prepared);

    log('Transaction ID:', signed.hash);

    const result = await client.submitAndWait(signed.tx_blob);

    await client.disconnect();

    console.log(JSON.stringify({
      success: true,
      data: {
        txHash: signed.hash,
        validated: result.result.validated,
        meta: result.result.meta,
      },
    }));
  } catch (error) {
    if (client.isConnected()) {
      await client.disconnect();
    }

    console.log(JSON.stringify({
      success: false,
      error: error.message,
      details: error.data ? JSON.stringify(error.data) : error.stack,
    }));
    process.exit(1);
  }
}

if (require.main === module) {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error('Usage: node clawback.js <config-json-path>');
    process.exit(1);
  }

  const configPath = args[0];
  if (!fs.existsSync(configPath)) {
    console.log(JSON.stringify({
      success: false,
      error: `Config file not found: ${configPath}`,
      details: 'Please provide a valid config JSON file path',
    }));
    process.exit(1);
  }

  const configContent = fs.readFileSync(configPath, 'utf8');
  clawbackContract(JSON.parse(configContent));
}

module.exports = { clawbackContract };
