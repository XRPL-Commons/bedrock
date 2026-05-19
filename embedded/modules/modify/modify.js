#!/usr/bin/env node

/**
 * XRPL Smart Contract Modify Module
 *
 * Handles modifying deployed contracts via ContractModify transaction.
 *
 * Usage: node modify.js <config-json-path>
 */

const xrpl = require('@xrpl-commons/xrpl');
const fs = require('fs');

function buildFunctionsFromABI(abi, exportedFunctions) {
  const functions = [];

  for (const fn of abi.functions) {
    if (!exportedFunctions.includes(fn.name)) {
      continue;
    }

    const parameters = fn.parameters.map((param) => ({
      Parameter: {
        ParameterFlag: param.flag,
        ParameterType: {
          type: param.type,
        },
      },
    }));

    functions.push({
      Function: {
        FunctionName: Buffer.from(fn.name).toString('hex').toUpperCase(),
        Parameters: parameters.length > 0 ? parameters : undefined,
      },
    });
  }

  return functions;
}

async function modifyContract(config) {
  const {
    contract_account,
    network_url,
    wallet_seed,
    algorithm,
    wasm_path,
    abi_path,
    fee,
    verbose,
    owner,
    contract_hash,
    immutable,
    code_immutable,
    abi_immutable,
    undeletable,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Modifying contract on XRPL...\n');

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
      TransactionType: 'ContractModify',
      Account: wallet.address,
      ContractAccount: contract_account,
      Fee: fee || '10000000',
    };

    // Optionally update WASM code (or reference by hash)
    if (contract_hash) {
      tx.ContractHash = contract_hash;
      log(`Referencing existing ContractSource: ${contract_hash}`);
    } else if (wasm_path && fs.existsSync(wasm_path)) {
      const wasmBytes = fs.readFileSync(wasm_path);
      tx.ContractCode = wasmBytes.toString('hex').toUpperCase();
      log(`Updated WASM: ${wasmBytes.length} bytes`);
    }

    // Optionally update ABI
    if (abi_path && fs.existsSync(abi_path)) {
      const abiContent = fs.readFileSync(abi_path, 'utf8');
      const abi = JSON.parse(abiContent);
      const functionNames = abi.functions.map(f => f.name);
      tx.Functions = buildFunctionsFromABI(abi, functionNames);
      log(`Updated ABI: ${abi.functions.length} functions`);
    }

    // Optionally transfer ownership
    if (owner) {
      tx.ContractOwner = owner;
      log(`Transferring ownership to: ${owner}`);
    }

    // Compute modify flags
    let modifyFlags = 0;
    if (immutable)      modifyFlags |= 0x00010000;
    if (code_immutable) modifyFlags |= 0x00020000;
    if (abi_immutable)  modifyFlags |= 0x00040000;
    if (undeletable)    modifyFlags |= 0x00080000;

    if (modifyFlags !== 0) {
      tx.Flags = modifyFlags;
      log(`Setting flags: 0x${modifyFlags.toString(16)}`);
    }

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
    console.error('Usage: node modify.js <config-json-path>');
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
  modifyContract(JSON.parse(configContent));
}

module.exports = { modifyContract };
