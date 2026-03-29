#!/usr/bin/env node

const xrpl = require('@transia/xrpl');

async function testConnection() {
  console.log('Testing connection to ws://localhost:6006...\n');

  const client = new xrpl.Client('ws://localhost:6006');

  try {
    console.log('Connecting...');
    await client.connect();
    console.log('✓ Connected successfully!\n');

    console.log('Requesting server info...');
    const serverInfo = await client.request({
      command: 'server_info'
    });
    console.log('✓ Server info received:');
    console.log('  Ledger:', serverInfo.result.info.validated_ledger?.seq || 'N/A');
    console.log('  State:', serverInfo.result.info.server_state);
    console.log('  Complete ledgers:', serverInfo.result.info.complete_ledgers);
    console.log();

    console.log('Requesting ledger info...');
    const ledgerInfo = await client.request({
      command: 'ledger',
      ledger_index: 'validated'
    });
    console.log('✓ Ledger info received:');
    console.log('  Index:', ledgerInfo.result.ledger_index);
    console.log('  Hash:', ledgerInfo.result.ledger_hash);
    console.log();

    await client.disconnect();
    console.log('✓ Test completed successfully!');

  } catch (error) {
    console.error('✗ Error:', error.message);
    console.error('Details:', error);
    if (client.isConnected()) {
      await client.disconnect();
    }
    process.exit(1);
  }
}

testConnection();
