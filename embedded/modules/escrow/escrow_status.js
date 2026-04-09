#!/usr/bin/env node

/**
 * XRPL Smart Escrow Status Module
 *
 * This module queries an escrow ledger entry on XRPL networks.
 *
 * Usage: node escrow_status.js <config-json-path>
 *
 * Config JSON format:
 * {
 *   "owner": "rXXX...",
 *   "escrow_sequence": 123,
 *   "network_url": "wss://alphanet.xrpl.org",
 *   "verbose": true (optional)
 * }
 *
 * Output JSON format:
 * {
 *   "success": true,
 *   "data": { ...escrow object fields... }
 * }
 */

const xrpl = require('xrpl');
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
 * Query escrow status from XRPL network
 */
async function getEscrowStatus(config) {
  const {
    owner,
    escrow_sequence,
    network_url,
    verbose,
  } = config;

  const log = verbose ? console.error.bind(console) : () => {};

  log('Querying escrow status on XRPL...\n');

  const isLocal = network_url.includes('localhost') || network_url.includes('127.0.0.1');

  try {
    let escrowData;

    if (isLocal) {
      // Local nodes: use HTTP RPC
      const rpcUrl = network_url
        .replace('ws://', 'http://').replace('wss://', 'https://')
        .replace('localhost:6006', 'localhost:5005');

      const response = await httpRPC(rpcUrl, 'ledger_entry', {
        escrow: {
          owner: owner,
          seq: escrow_sequence,
        },
      }, 30000);

      log('Ledger entry response received');
      escrowData = response.node || response;
    } else {
      // Remote networks: use WebSocket
      const client = new xrpl.Client(network_url);
      client.apiVersion = 1;

      try {
        await client.connect();
        log('Connected to network');

        const response = await client.request({
          command: 'ledger_entry',
          escrow: {
            owner: owner,
            seq: escrow_sequence,
          },
        });

        log('Ledger entry response received');
        escrowData = response.result.node || response.result;

        await client.disconnect();
      } catch (innerError) {
        if (client.isConnected()) {
          await client.disconnect();
        }
        throw innerError;
      }
    }

    log('\nEscrow data retrieved successfully!');

    const result = {
      success: true,
      data: escrowData,
    };

    console.log(JSON.stringify(result));
    return result;
  } catch (error) {
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
Usage: node escrow_status.js <config-json-path>

The config JSON file should contain:
{
  "owner": "rXXX...",
  "escrow_sequence": 123,
  "network_url": "wss://alphanet.xrpl.org",
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
    getEscrowStatus(config);
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

module.exports = { getEscrowStatus };
