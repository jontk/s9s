const { chromium } = require('playwright');

// Test configuration  
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';
const WAIT_TIME = 1500; // milliseconds between actions

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testJobsTableWidth() {
  const browser = await chromium.launch({ 
    headless: false, // Set to true for CI/CD
    slowMo: 100 // Slow down actions to see what's happening
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('📏 Testing Jobs table width expansion...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000); // Wait for terminal to initialize
    
    // Test Jobs view table width
    console.log('📊 Testing Jobs view (1)...');
    await page.keyboard.press('1');
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/jobs-width-after-fix.png', 
      fullPage: true 
    });
    
    // Test Nodes view for comparison  
    console.log('📊 Testing Nodes view (2)...');
    await page.keyboard.press('2');
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/nodes-width-comparison.png', 
      fullPage: true 
    });
    
    // Test Partitions view for comparison
    console.log('📊 Testing Partitions view (3)...');
    await page.keyboard.press('3');
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/partitions-width-comparison.png', 
      fullPage: true 
    });
    
    // Go back to Jobs and test multi-select with expanded width
    console.log('📊 Testing Jobs multi-select with full width...');
    await page.keyboard.press('1');
    await sleep(WAIT_TIME);
    
    // Enable multi-select mode
    await page.keyboard.press('v');
    await sleep(WAIT_TIME);
    
    // Navigate and select some rows to test width with checkboxes
    await page.keyboard.press('ArrowDown');
    await sleep(300);
    await page.keyboard.press(' '); // Toggle selection
    await sleep(300);
    await page.keyboard.press('ArrowDown');
    await sleep(300);
    await page.keyboard.press(' '); // Toggle another row
    await sleep(300);
    
    await page.screenshot({ 
      path: 'test/e2e/jobs-multiselect-width-after-fix.png', 
      fullPage: true 
    });
    
    // Turn off multi-select
    await page.keyboard.press('v');
    await sleep(WAIT_TIME);
    
    console.log('✅ Jobs table width test completed!');
    console.log('📸 Screenshots saved - compare jobs-width-after-fix.png with nodes/partitions');
    console.log('🔍 Jobs table should now expand to full width like other views');
    
  } catch (error) {
    console.error('❌ Jobs width test failed:', error);
    await page.screenshot({ path: 'test/e2e/jobs-width-error.png', fullPage: true });
    throw error;
  } finally {
    await browser.close();
  }
}

// Run the test
testJobsTableWidth().catch(console.error);