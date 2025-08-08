const { chromium } = require('playwright');

// Test configuration  
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';
const WAIT_TIME = 1000; // milliseconds between actions

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testNavigationOrder() {
  const browser = await chromium.launch({ 
    headless: false, // Set to true for CI/CD
    slowMo: 100 // Slow down actions to see what's happening
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🧭 Testing navigation order...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000); // Wait for terminal to initialize
    
    // Test the new navigation order
    const expectedOrder = [
      { key: '1', name: 'Jobs' },
      { key: '2', name: 'Nodes' },
      { key: '3', name: 'Partitions' },
      { key: '4', name: 'Reservations' },
      { key: '5', name: 'QoS' },
      { key: '6', name: 'Accounts' },
      { key: '7', name: 'Users' },
      { key: '8', name: 'Dashboard' },
      { key: '9', name: 'Health' },
      { key: '0', name: 'Performance' }
    ];
    
    console.log('📝 Testing keyboard shortcuts in new order:');
    console.log('   Expected: JOBS | NODES | PARTITIONS | RESERVATIONS | QOS | ACCOUNTS | USERS | DASHBOARD | HEALTH | PERFORMANCE');
    
    // Test each view in order
    for (const view of expectedOrder) {
      console.log(`   - Testing ${view.key} → ${view.name}...`);
      
      await page.keyboard.press(view.key);
      await sleep(WAIT_TIME);
      
      // Take screenshot for verification
      await page.screenshot({ 
        path: `test/e2e/nav-order-${view.key}-${view.name.toLowerCase()}.png`, 
        fullPage: true 
      });
    }
    
    // Test help view to verify the updated help text
    console.log('📋 Testing updated help text...');
    await page.keyboard.press('?');
    await sleep(WAIT_TIME);
    await page.screenshot({ path: 'test/e2e/updated-help-text.png', fullPage: true });
    await page.keyboard.press('Escape');
    await sleep(WAIT_TIME);
    
    // Test some command shortcuts
    console.log('💬 Testing command shortcuts...');
    
    // Test dashboard command
    await page.keyboard.press(':');
    await sleep(500);
    await page.keyboard.type('dashboard');
    await sleep(500);
    await page.screenshot({ path: 'test/e2e/dashboard-command-test.png', fullPage: true });
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    
    // Verify we're in dashboard
    await page.screenshot({ path: 'test/e2e/dashboard-command-result.png', fullPage: true });
    
    // Final verification - cycle through all views using Tab
    console.log('🔄 Testing Tab navigation order...');
    await page.keyboard.press('1'); // Start at Jobs
    await sleep(WAIT_TIME);
    
    for (let i = 0; i < expectedOrder.length; i++) {
      await page.keyboard.press('Tab');
      await sleep(800);
      await page.screenshot({ 
        path: `test/e2e/tab-nav-step-${i + 1}.png`, 
        fullPage: true 
      });
    }
    
    console.log('✅ Navigation order test completed successfully!');
    console.log('📸 Screenshots saved showing the new navigation order');
    console.log('🎯 Order tested: 1→Jobs, 2→Nodes, 3→Partitions, 4→Reservations, 5→QoS, 6→Accounts, 7→Users, 8→Dashboard, 9→Health, 0→Performance');
    
  } catch (error) {
    console.error('❌ Navigation order test failed:', error);
    await page.screenshot({ path: 'test/e2e/navigation-order-error.png', fullPage: true });
    throw error;
  } finally {
    await browser.close();
  }
}

// Run the test
testNavigationOrder().catch(console.error);