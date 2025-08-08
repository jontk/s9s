const { chromium } = require('playwright');

// Test configuration
// Default to localhost for ttyd, but allow override via environment variable
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';
const WAIT_TIME = 1000; // milliseconds between actions

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testS9sTUI() {
  const browser = await chromium.launch({ 
    headless: false, // Set to true for CI/CD
    slowMo: 100 // Slow down actions to see what's happening
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('1. Navigating to ttyd...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(2000); // Wait for terminal to initialize
    
    // The ttyd terminal is inside the body element
    console.log('2. Testing navigation between views...');
    
    // Test Jobs view (default)
    console.log('   - Jobs view (j)');
    await page.keyboard.press('j');
    await sleep(WAIT_TIME);
    
    // Test Nodes view
    console.log('   - Nodes view (n)');
    await page.keyboard.press('n');
    await sleep(WAIT_TIME);
    
    // Test Partitions view
    console.log('   - Partitions view (p)');
    await page.keyboard.press('p');
    await sleep(WAIT_TIME);
    
    // Test Users view
    console.log('   - Users view (u)');
    await page.keyboard.press('u');
    await sleep(WAIT_TIME);
    
    // Test Help view
    console.log('   - Help view (?)');
    await page.keyboard.press('?');
    await sleep(WAIT_TIME);
    
    // Exit help with ESC
    await page.keyboard.press('Escape');
    await sleep(WAIT_TIME);
    
    // Go back to Jobs view
    console.log('3. Testing Jobs view functionality...');
    await page.keyboard.press('j');
    await sleep(WAIT_TIME);
    
    // Test navigation
    console.log('   - Navigate down');
    await page.keyboard.press('ArrowDown');
    await sleep(500);
    await page.keyboard.press('ArrowDown');
    await sleep(500);
    
    console.log('   - Navigate up');
    await page.keyboard.press('ArrowUp');
    await sleep(500);
    
    // Test filter
    console.log('   - Open filter (/)');
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.type('test');
    await sleep(500);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    
    // Clear filter
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.press('Control+U'); // Clear line
    await sleep(500);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    
    // Test auto-refresh toggle
    console.log('   - Toggle auto-refresh (m)');
    await page.keyboard.press('m');
    await sleep(WAIT_TIME);
    await page.keyboard.press('m'); // Toggle back
    await sleep(WAIT_TIME);
    
    // Test batch operations
    console.log('   - Open batch operations (b)');
    await page.keyboard.press('b');
    await sleep(WAIT_TIME);
    await page.keyboard.press('Escape'); // Close menu
    await sleep(WAIT_TIME);
    
    // Test refresh
    console.log('   - Refresh (R)');
    await page.keyboard.press('R');
    await sleep(2000);
    
    // Test Nodes view operations
    console.log('4. Testing Nodes view functionality...');
    await page.keyboard.press('n');
    await sleep(WAIT_TIME);
    
    // Navigate in nodes
    await page.keyboard.press('ArrowDown');
    await sleep(500);
    
    // Test SSH (if available)
    console.log('   - Try SSH (s) - will fail if no node selected');
    await page.keyboard.press('s');
    await sleep(WAIT_TIME);
    
    // Test Dashboard view
    console.log('5. Testing Dashboard view...');
    await page.keyboard.press('d');
    await sleep(2000);
    
    // Test Settings
    console.log('6. Testing Settings view...');
    await page.keyboard.press('S');
    await sleep(WAIT_TIME);
    await page.keyboard.press('Escape'); // Exit settings
    await sleep(WAIT_TIME);
    
    // Take screenshot
    console.log('7. Taking screenshot...');
    await page.screenshot({ path: 'test/e2e/s9s-ttyd-test.png', fullPage: true });
    
    // Test exit
    console.log('8. Testing exit (q)');
    await page.keyboard.press('q');
    await sleep(1000);
    await page.keyboard.press('y'); // Confirm exit
    await sleep(1000);
    
    console.log('✅ All tests completed successfully!');
    
  } catch (error) {
    console.error('❌ Test failed:', error);
    await page.screenshot({ path: 'test/e2e/s9s-ttyd-error.png', fullPage: true });
    throw error;
  } finally {
    await browser.close();
  }
}

// Run the test
testS9sTUI().catch(console.error);