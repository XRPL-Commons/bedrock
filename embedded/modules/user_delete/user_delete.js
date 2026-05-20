#!/usr/bin/env node

/**
 * XRPL Contract User Delete Module
 *
 * Handles deleting user's data from a contract via ContractUserDelete transaction.
 * This recovers the user's reserves.
 *
 * Usage: node user_delete.js <config-json-path>
 */

const xrpl = require('@xrpl-commons/xrpl');
const fs = require('fs');

async function contractUserDelete(config) {
  const {
    contract_account,
    network_url,
    wallet_seed,
    algorithm,
    fee,
    verbose,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Deleting user data from contract...\n');

  const client = new xrpl.Client(network_url);
  client.apiVersion = 1;

  try {
    await client.connect();
    log('Connected to network');

    const algo = algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    const wallet = algo
      ? xrpl.Wallet.fromSeed(wallet_seed, { algorithm: algo })
      : xrpl.Wallet.fromSeed(wallet_seed);

    const tx = {
      TransactionType: 'ContractUserDelete',
      Account: wallet.address,
      ContractAccount: contract_account,
      Fee: fee || '1000000',
    };

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
    console.error('Usage: node user_delete.js <config-json-path>');
    process.exit(1);
  }

  const configPath = args[0];
  let configContent;
  if (!configPath || configPath === '-') {
    // Config piped via stdin; reading fd 0 synchronously is supported in Node 12+.
    configContent = fs.readFileSync(0, 'utf8');
  } else if (fs.existsSync(configPath)) {
    configContent = fs.readFileSync(configPath, 'utf8');
  } else {
    console.log(JSON.stringify({
      success: false,
      error: `Config file not found: ${configPath}`,
      details: 'Provide config via stdin (recommended) or as a valid file path',
    }));
    process.exit(1);
  }
  contractUserDelete(JSON.parse(configContent));
}

module.exports = { contractUserDelete };
