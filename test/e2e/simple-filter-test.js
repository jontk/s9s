const { chromium } = require('playwright');

// Test configuration
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testSimpleFilter() {
  const browser = await chromium.launch({ 
    headless: false,
    slowMo: 200
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🔍 Simple filter test...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000);
    
    // Go to Jobs view
    console.log('📊 Going to Jobs view...');
    await page.keyboard.press('1');
    await sleep(2000);
    await page.screenshot({ path: 'test/e2e/simple-jobs-before.png' });
    
    // Try to focus filter - press '/' and wait to see what happens
    console.log('🔍 Pressing / to focus filter...');
    await page.keyboard.press('/');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/simple-after-slash.png' });
    
    // If no modal appeared, try typing filter text
    console.log('⌨️  Typing "sleep"...');  
    await page.keyboard.type('sleep');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/simple-after-typing.png' });
    
    // Press Enter to apply filter
    console.log('⏎ Pressing Enter...');
    await page.keyboard.press('Enter');
    await sleep(2000);
    await page.screenshot({ path: 'test/e2e/simple-after-enter.png' });
    
    console.log('✅ Simple filter test completed!');
    
  } catch (error) {
    console.error('❌ Simple filter test failed:', error);
    await page.screenshot({ path: 'test/e2e/simple-filter-error.png' });
    throw error;
  } finally {
    await browser.close();
  }
}

testSimpleFilter().catch(console.error);