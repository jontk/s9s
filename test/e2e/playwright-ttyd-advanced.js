const { chromium } = require('playwright');

// Test configuration
// Default to localhost for ttyd, but allow override via environment variable
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';

// Helper functions
async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function takeScreenshot(page, name) {
  await page.screenshot({ 
    path: `test/e2e/screenshots/${name}.png`, 
    fullPage: true 
  });
  console.log(`   📸 Screenshot saved: ${name}.png`);
}

async function testJobOperations(page) {
  console.log('\n🧪 Testing Job Operations...');
  
  // Navigate to Jobs view
  await page.keyboard.press('j');
  await sleep(1000);
  
  // Select a job
  await page.keyboard.press('ArrowDown');
  await sleep(500);
  
  // Test job details (Enter)
  console.log('   - View job details (Enter)');
  await page.keyboard.press('Enter');
  await sleep(1000);
  await takeScreenshot(page, 'job-details');
  await page.keyboard.press('Escape');
  await sleep(500);
  
  // Test job output
  console.log('   - View job output (o)');
  await page.keyboard.press('o');
  await sleep(1000);
  
  // If output viewer opens, close it
  await page.keyboard.press('Escape');
  await sleep(500);
  
  // Test state filters
  console.log('   - Filter by running jobs (Alt+r)');
  await page.keyboard.press('Alt+r');
  await sleep(1000);
  
  console.log('   - Show all jobs (a)');
  await page.keyboard.press('a');
  await sleep(1000);
}

async function testNodeOperations(page) {
  console.log('\n🧪 Testing Node Operations...');
  
  // Navigate to Nodes view
  await page.keyboard.press('n');
  await sleep(1000);
  await takeScreenshot(page, 'nodes-view');
  
  // Select a node
  await page.keyboard.press('ArrowDown');
  await sleep(500);
  
  // Test node details
  console.log('   - View node details (Enter)');
  await page.keyboard.press('Enter');
  await sleep(1000);
  await takeScreenshot(page, 'node-details');
  await page.keyboard.press('Escape');
  await sleep(500);
  
  // Test node state filter
  console.log('   - Toggle node state display (t)');
  await page.keyboard.press('t');
  await sleep(1000);
}

async function testKeyboardShortcuts(page) {
  console.log('\n🧪 Testing Keyboard Shortcuts...');
  
  // Test quick navigation
  const views = [
    { key: 'j', name: 'Jobs' },
    { key: 'n', name: 'Nodes' },
    { key: 'p', name: 'Partitions' },
    { key: 'r', name: 'Reservations' },
    { key: 'u', name: 'Users' },
    { key: 'a', name: 'Accounts' },
    { key: 'q', name: 'QoS' },
    { key: 'd', name: 'Dashboard' }
  ];
  
  for (const view of views) {
    console.log(`   - Switch to ${view.name} view (${view.key})`);
    await page.keyboard.press(view.key);
    await sleep(800);
  }
  
  // Test global shortcuts
  console.log('   - Show help (?)');
  await page.keyboard.press('?');
  await sleep(1000);
  await takeScreenshot(page, 'help-screen');
  await page.keyboard.press('Escape');
  await sleep(500);
  
  console.log('   - Refresh current view (R)');
  await page.keyboard.press('R');
  await sleep(1500);
}

async function testSearch(page) {
  console.log('\n🧪 Testing Search Functionality...');
  
  // Go to Jobs view
  await page.keyboard.press('j');
  await sleep(1000);
  
  // Open search
  console.log('   - Open search (/)');
  await page.keyboard.press('/');
  await sleep(500);
  
  // Type search term
  console.log('   - Search for "test"');
  await page.keyboard.type('test');
  await sleep(500);
  await takeScreenshot(page, 'search-active');
  
  // Apply search
  await page.keyboard.press('Enter');
  await sleep(1000);
  
  // Clear search
  console.log('   - Clear search');
  await page.keyboard.press('/');
  await sleep(500);
  await page.keyboard.press('Control+U');
  await sleep(500);
  await page.keyboard.press('Enter');
  await sleep(1000);
}

async function testAdvancedFilters(page) {
  console.log('\n🧪 Testing Advanced Filters...');
  
  // Open advanced filter
  console.log('   - Open advanced filter (F3)');
  await page.keyboard.press('F3');
  await sleep(1000);
  
  // Close if opened
  await page.keyboard.press('Escape');
  await sleep(500);
}

async function runTests() {
  const browser = await chromium.launch({ 
    headless: false,
    slowMo: 50,
    devtools: false
  });
  
  const context = await browser.newContext({
    viewport: { width: 1400, height: 900 },
    recordVideo: {
      dir: 'test/e2e/videos',
      size: { width: 1400, height: 900 }
    }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🚀 Starting s9s TUI tests via ttyd...\n');
    
    // Navigate to ttyd
    console.log('📍 Navigating to', TTYD_URL);
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000); // Wait for terminal to fully initialize
    
    // Take initial screenshot
    await takeScreenshot(page, 'initial-load');
    
    // Run test suites
    await testKeyboardShortcuts(page);
    await testJobOperations(page);
    await testNodeOperations(page);
    await testSearch(page);
    await testAdvancedFilters(page);
    
    console.log('\n✅ All tests completed successfully!');
    
    // Final screenshot
    await takeScreenshot(page, 'final-state');
    
  } catch (error) {
    console.error('\n❌ Test failed:', error);
    await takeScreenshot(page, 'error-state');
    throw error;
  } finally {
    // Save video if recorded
    const video = page.video();
    if (video) {
      await video.saveAs('test/e2e/videos/s9s-tui-test.webm');
      console.log('\n📹 Test video saved: test/e2e/videos/s9s-tui-test.webm');
    }
    
    await context.close();
    await browser.close();
  }
}

// Create directories if they don't exist
const fs = require('fs');
const path = require('path');

const dirs = ['test/e2e/screenshots', 'test/e2e/videos'];
dirs.forEach(dir => {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
});

// Run the tests
runTests().catch(console.error);