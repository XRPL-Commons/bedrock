#!/usr/bin/env node

/**
 * XRPL Smart Contract Call Module
 *
 * This module handles calling functions on deployed XRPL smart contracts.
 * It uses the ABI to construct properly-typed parameter values.
 *
 * Usage: node call.js <config-json-path>
 *
 * Config JSON format:
 * {
 *   "contract_account": "rContract123...",
 *   "function_name": "register",
 *   "network_url": "wss://alphanet.xrpl.org",
 *   "wallet_seed": "sXXX...",
 *   "abi_path": "/path/to/abi.json" (optional),
 *   "parameters": {"name": "test", "duration": 31536000} (optional),
 *   "computation_allowance": "1000000" (optional),
 *   "fee": "1000000" (optional),
 *   "verbose": true (optional)
 * }
 *
 * Output JSON format:
 * {
 *   "success": true,
 *   "data": {
 *     "txHash": "...",
 *     "returnCode": 0,
 *     "returnValue": "...",
 *     "gasUsed": 12345,
 *     "validated": true
 *   }
 * }
 */

const xrpl = require('xrpl');
const fs = require('fs');
const http = require('http');
const https = require('https');

/**
 * Build Parameters array from ABI and provided values
 */
function buildParametersFromABI(functionDef, paramValues) {
  if (!functionDef.parameters || functionDef.parameters.length === 0) {
    return undefined;
  }

  const parameters = [];

  for (let i = 0; i < functionDef.parameters.length; i++) {
    const paramDef = functionDef.parameters[i];
    const value = paramValues[paramDef.name] || paramValues[i];

    if (value === undefined && paramDef.flag === 0) {
      throw new Error(
        `Required parameter '${paramDef.name}' (${paramDef.type}) not provided`
      );
    }

    if (value !== undefined) {
      parameters.push({
        ParameterFlag: paramDef.flag,
        ParameterValue: formatParameterValue(paramDef.type, value),
      });
    }
  }

  return parameters.length > 0 ? parameters : undefined;
}

/**
 * Format parameter value according to XRPL type
 */
function formatParameterValue(type, value) {
  switch (type) {
    case 'UINT8':
    case 'UINT16':
    case 'UINT32':
    case 'UINT64':
    case 'UINT128':
    case 'UINT160':
    case 'UINT192':
    case 'UINT256':
      return {
        type: type,
        value: value.toString(),
      };

    case 'VL':
      // Variable length - convert string to hex
      return {
        type: 'VL',
        value:
          typeof value === 'string' && !value.startsWith('0x')
            ? Buffer.from(value).toString('hex').toUpperCase()
            : value.replace('0x', '').toUpperCase(),
      };

    case 'ACCOUNT':
      return {
        type: 'ACCOUNT',
        value: value, // r-address format
      };

    case 'AMOUNT':
      // XRP drops or IOU
      if (typeof value === 'string' || typeof value === 'number') {
        return {
          type: 'AMOUNT',
          value: value.toString(),
        };
      }
      return {
        type: 'AMOUNT',
        value: value, // Already formatted object
      };

    case 'NUMBER':
      return {
        type: 'NUMBER',
        value: parseFloat(value),
      };

    case 'CURRENCY':
      return {
        type: 'CURRENCY',
        value: value,
      };

    case 'ISSUE':
      return {
        type: 'ISSUE',
        value: value,
      };

    default:
      throw new Error(`Unsupported parameter type: ${type}`);
  }
}

/**
 * Call a contract function
 */
async function callContract(config) {
  const {
    contract_account,
    function_name,
    network_url,
    wallet_seed,
    abi_path,
    parameters,
    computation_allowance,
    fee,
    verbose,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Calling smart contract on XRPL...\n');

  const client = new xrpl.Client(network_url);
  client.apiVersion = 1;

  try {
    await client.connect();
    log('✓ Connected to network');

    // Create or restore wallet
    const algorithm = config.algorithm === 'ed25519' ? undefined : xrpl.ECDSA.secp256k1;
    const wallet = wallet_seed
      ? (algorithm ? xrpl.Wallet.fromSeed(wallet_seed, { algorithm }) : xrpl.Wallet.fromSeed(wallet_seed))
      : (algorithm ? xrpl.Wallet.generate(algorithm) : xrpl.Wallet.generate());

    log('\nWallet:');
    log('  Address:', wallet.address);

    log(`\nContract: ${contract_account}`);
    log(`Function: ${function_name}`);

    // Load ABI if provided
    let functionDef = null;
    let Parameters = undefined;

    if (abi_path && fs.existsSync(abi_path)) {
      log(`\nLoading ABI from: ${abi_path}`);
      const abiContent = fs.readFileSync(abi_path, 'utf8');
      const abi = JSON.parse(abiContent);

      // Find function in ABI
      functionDef = abi.functions.find((f) => f.name === function_name);

      if (functionDef) {
        log(`\nFunction signature:`);
        log(`  ${function_name}(`);
        if (functionDef.parameters) {
          functionDef.parameters.forEach((p) => {
            const required = p.flag === 0 ? 'required' : 'optional';
            log(`    ${p.name}: ${p.type} (${required})`);
          });
        }
        log(`  )`);
        if (functionDef.returns) {
          log(`  -> ${functionDef.returns.type}`);
        }

        // Build parameters from ABI
        if (parameters) {
          log(`\nProvided parameters:`);
          log(JSON.stringify(parameters, null, 2));

          Parameters = buildParametersFromABI(functionDef, parameters);

          if (Parameters) {
            log(`\nFormatted parameters:`);
            log(JSON.stringify(Parameters, null, 2));
          }
        }
      } else {
        log(`\nWarning: Function "${function_name}" not found in ABI`);
      }
    }

    // Check balance
    const balance = await client.getXrpBalance(wallet.address);
    log(`\nWallet balance: ${balance} XRP`);

    if (parseFloat(balance) === 0) {
      log('Warning: Wallet not funded, call will likely fail');
    }

    // Create ContractCall transaction
    log('\nSubmitting contract call transaction...');

    const functionNameHex = Buffer.from(function_name)
      .toString('hex')
      .toUpperCase();

    // Manual construction: autofill tries to simulate ContractCall which nodes may not support
    const accountInfo = await client.request({
      command: 'account_info',
      account: wallet.address,
    });

    const tx = {
      TransactionType: 'ContractCall',
      Account: wallet.address,
      ContractAccount: contract_account,
      FunctionName: functionNameHex,
      Parameters: Parameters,
      ComputationAllowance: parseInt(computation_allowance || '1000000'),
      Fee: fee || '1000000',
      Sequence: accountInfo.result.account_data.Sequence,
      SigningPubKey: wallet.publicKey,
    };

    if (config.network_id && config.network_id > 1024) {
      tx.NetworkID = config.network_id;
    }

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

    log('\n✓ Contract function called successfully!');

    const meta = txResult.meta;

    log('\nTransaction Status:');
    log(`  Result: ${meta?.TransactionResult || 'N/A'}`);
    log(`  Validated: ${txResult.validated ? 'Yes' : 'No'}`);

    if (meta?.WasmReturnCode !== undefined) {
      log('\nContract Return Code:');
      log(`  Value: ${meta.WasmReturnCode}`);
      log(`  Status: ${meta.WasmReturnCode === 0 ? 'SUCCESS' : 'ERROR'}`);
    }

    if (meta?.ReturnValue) {
      log('\nContract Return Data:');
      log(`  Hex: ${meta.ReturnValue}`);
      log(`  Decimal: ${parseInt(meta.ReturnValue, 16)}`);

      // Try to decode based on ABI return type
      if (functionDef?.returns) {
        log(`  Type: ${functionDef.returns.type}`);
        log(`  Description: ${functionDef.returns.description || 'N/A'}`);
      }
    }

    if (meta?.GasUsed !== undefined) {
      log('\nGas/Computation Used:');
      log(`  Gas Used: ${meta.GasUsed}`);
      log(`  Allowance: ${tx.ComputationAllowance}`);
      const percentage = (
        (meta.GasUsed / parseInt(tx.ComputationAllowance)) *
        100
      ).toFixed(2);
      log(`  Percentage: ${percentage}%`);
    }

    // Output call result as clean JSON to stdout
    const callResult = {
      success: true,
      data: {
        txHash: signed.hash,
        returnCode: meta?.WasmReturnCode,
        returnValue: meta?.ReturnValue,
        gasUsed: meta?.GasUsed,
        validated: txResult.validated,
        transactionResult: meta?.TransactionResult,
        meta: meta,
      },
    };

    console.log(JSON.stringify(callResult));
    return callResult;
  } catch (error) {
    if (client.isConnected()) {
      await client.disconnect();
    }

    // Output error as JSON
    const errorResult = {
      success: false,
      error: error.message,
      details: error.data ? JSON.stringify(error.data) : error.stack,
    };

    console.log(JSON.stringify(errorResult));
    process.exit(1);
  }
}

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

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error(`
Usage: node call.js <config-json-path>

The config JSON file should contain:
{
  "contract_account": "rContract123...",
  "function_name": "register",
  "network_url": "wss://alphanet.xrpl.org",
  "wallet_seed": "sXXX...",
  "abi_path": "/path/to/abi.json" (optional),
  "parameters": {"name": "test", "duration": 31536000} (optional),
  "computation_allowance": "1000000" (optional),
  "fee": "1000000" (optional),
  "verbose": true (optional)
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
    callContract(config);
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

module.exports = { callContract, buildParametersFromABI, formatParameterValue };
