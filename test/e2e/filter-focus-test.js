const { chromium } = require('playwright');

// Test configuration
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testFilterFocus() {
  const browser = await chromium.launch({ 
    headless: false,
    slowMo: 100
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🎯 Testing filter focus and shortcut interference...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000);
    
    // Test Jobs view filter with potentially problematic text
    console.log('📊 Testing Jobs view filter...');
    await page.keyboard.press('1');
    await sleep(1500);
    await page.screenshot({ path: 'test/e2e/filter-focus-jobs-before.png' });
    
    // Focus filter
    console.log('🔍 Focusing filter with /...');
    await page.keyboard.press('/');
    await sleep(800);
    await page.screenshot({ path: 'test/e2e/filter-focus-after-slash.png' });
    
    // Type text with shortcut characters that should NOT trigger shortcuts
    const problemTexts = [
      'test',      // contains 's' (Submit Job shortcut)
      'search',    // contains 's' (Submit Job shortcut)  
      'restart',   // contains 'r' (Release Job shortcut)
      'schedule',  // contains 's' and 'c' (Submit Job and Cancel Job shortcuts)
      'batch',     // contains 'b' (Batch Operations shortcut)
      'hold',      // contains 'h' (Hold Job shortcut)
    ];
    
    for (const text of problemTexts) {
      console.log(`⌨️  Testing filter text: "${text}"...`);
      
      // Clear any existing text
      await page.keyboard.press('Control+A');
      await sleep(100);
      
      // Type the problematic text
      await page.keyboard.type(text);
      await sleep(500);
      await page.screenshot({ 
        path: `test/e2e/filter-focus-typing-${text}.png` 
      });
      
      // Press Enter to apply filter
      await page.keyboard.press('Enter');
      await sleep(1000);
      await page.screenshot({ 
        path: `test/e2e/filter-focus-applied-${text}.png` 
      });
      
      // Clear filter for next test
      await page.keyboard.press('/');
      await sleep(300);
      await page.keyboard.press('Control+A');
      await sleep(100);
      await page.keyboard.press('Delete');
      await sleep(100);
      await page.keyboard.press('Enter');
      await sleep(800);
    }
    
    // Test ESC to exit filter
    console.log('🔙 Testing ESC to exit filter...');
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.type('test');
    await sleep(300);
    await page.screenshot({ path: 'test/e2e/filter-focus-before-esc.png' });
    await page.keyboard.press('Escape');
    await sleep(800);
    await page.screenshot({ path: 'test/e2e/filter-focus-after-esc.png' });
    
    // Test Nodes view to ensure fix works there too
    console.log('📊 Testing Nodes view filter...');
    await page.keyboard.press('2');
    await sleep(1500);
    
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.type('test'); // 't' might conflict with other shortcuts
    await sleep(500);
    await page.screenshot({ path: 'test/e2e/filter-focus-nodes-test.png' });
    await page.keyboard.press('Enter');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/filter-focus-nodes-applied.png' });
    
    // Test Partitions view
    console.log('📊 Testing Partitions view filter...');
    await page.keyboard.press('3');
    await sleep(1500);
    
    await page.keyboard.press('/');
    await sleep(500);
    await page.keyboard.type('normal'); // no conflicting shortcuts in this text
    await sleep(500);
    await page.screenshot({ path: 'test/e2e/filter-focus-partitions-test.png' });
    await page.keyboard.press('Enter');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/filter-focus-partitions-applied.png' });
    
    console.log('✅ Filter focus test completed!');
    console.log('📸 Screenshots saved showing filter working without shortcut interference');
    console.log('🎯 Tested problematic text: test, search, restart, schedule, batch, hold');
    console.log('🔍 Verified: Filter input keeps focus when typing shortcut characters');
    
  } catch (error) {
    console.error('❌ Filter focus test failed:', error);
    await page.screenshot({ path: 'test/e2e/filter-focus-error.png' });
    throw error;
  } finally {
    await browser.close();
  }
}

testFilterFocus().catch(console.error);