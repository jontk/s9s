const { chromium } = require('playwright');

// Test configuration
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';
const WAIT_TIME = 1500; // milliseconds between actions

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testTableExpansion() {
  const browser = await chromium.launch({ 
    headless: false, // Set to true for CI/CD
    slowMo: 50 // Slow down actions to see what's happening
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🔍 Testing table expansion across all views...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000); // Wait for terminal to initialize and s9s to start
    
    // Test each view and check if tables expand properly
    const views = [
      { key: '1', name: 'Jobs', shortcut: 'j' },
      { key: '2', name: 'Nodes', shortcut: 'n' }, 
      { key: '3', name: 'Partitions', shortcut: 'p' },
      { key: '4', name: 'Reservations', shortcut: 'r' },
      { key: '5', name: 'QoS', shortcut: 'q' },
      { key: '6', name: 'Accounts', shortcut: 'a' },
      { key: '7', name: 'Users', shortcut: 'u' },
      { key: '8', name: 'Dashboard', shortcut: 'd' },
      { key: '9', name: 'Health', shortcut: 'h' },
      { key: '0', name: 'Performance', shortcut: 'P' }
    ];
    
    console.log(`📏 Testing table expansion in ${views.length} views...`);
    
    for (const view of views) {
      console.log(`   - Testing ${view.name} view (${view.key})...`);
      
      // Switch to view using number key
      await page.keyboard.press(view.key);
      await sleep(WAIT_TIME);
      
      // Take a screenshot for this view
      await page.screenshot({ 
        path: `test/e2e/table-expansion-${view.name.toLowerCase()}.png`, 
        fullPage: true 
      });
      
      // Try to navigate within the table to ensure it's functional
      await page.keyboard.press('ArrowDown');
      await sleep(200);
      await page.keyboard.press('ArrowUp');
      await sleep(200);
    }
    
    // Specifically test Jobs view table expansion
    console.log('🎯 Detailed Jobs view table expansion test...');
    await page.keyboard.press('1'); // Go to Jobs view
    await sleep(WAIT_TIME);
    
    // Test multi-select functionality that was part of the fix
    console.log('   - Testing multi-select mode (v)...');
    await page.keyboard.press('v');
    await sleep(WAIT_TIME);
    
    // Navigate and select some rows
    await page.keyboard.press('ArrowDown');
    await sleep(300);
    await page.keyboard.press(' '); // Toggle selection with space
    await sleep(300);
    await page.keyboard.press('ArrowDown');
    await sleep(300);
    await page.keyboard.press(' '); // Toggle another row
    await sleep(300);
    
    // Take screenshot showing multi-select state
    await page.screenshot({ 
      path: 'test/e2e/jobs-multiselect-expansion.png', 
      fullPage: true 
    });
    
    // Turn off multi-select mode
    await page.keyboard.press('v');
    await sleep(WAIT_TIME);
    
    // Compare Jobs view with Nodes view side by side
    console.log('🔄 Comparing Jobs vs Nodes table expansion...');
    
    // Jobs view
    await page.keyboard.press('1');
    await sleep(WAIT_TIME);
    await page.screenshot({ path: 'test/e2e/comparison-jobs.png', fullPage: true });
    
    // Nodes view  
    await page.keyboard.press('2');
    await sleep(WAIT_TIME);
    await page.screenshot({ path: 'test/e2e/comparison-nodes.png', fullPage: true });
    
    // Test filter functionality in both views to ensure layouts work
    console.log('🔍 Testing filter functionality...');
    
    // Test filter in Jobs view
    await page.keyboard.press('1');
    await sleep(WAIT_TIME);
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.type('test');
    await sleep(500);
    await page.screenshot({ path: 'test/e2e/jobs-filter-test.png', fullPage: true });
    await page.keyboard.press('Escape');
    await sleep(500);
    
    // Test filter in Nodes view
    await page.keyboard.press('2');
    await sleep(WAIT_TIME);
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.type('compute');
    await sleep(500);
    await page.screenshot({ path: 'test/e2e/nodes-filter-test.png', fullPage: true });
    await page.keyboard.press('Escape');
    await sleep(500);
    
    // Final comprehensive screenshot
    await page.keyboard.press('1'); // Back to Jobs view for final test
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/final-jobs-table-expansion.png', 
      fullPage: true 
    });
    
    console.log('✅ Table expansion test completed successfully!');
    console.log('📸 Screenshots saved in test/e2e/ directory');
    console.log('🔎 Please review the screenshots to verify table expansion is consistent across views');
    
  } catch (error) {
    console.error('❌ Table expansion test failed:', error);
    await page.screenshot({ path: 'test/e2e/table-expansion-error.png', fullPage: true });
    throw error;
  } finally {
    await browser.close();
  }
}

// Run the test
testTableExpansion().catch(console.error);