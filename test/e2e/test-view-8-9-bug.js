const { chromium } = require('playwright');

/**
 * Test script to reproduce the view corruption bug when switching
 * between Health (8) and Performance (9) views in s9s TUI
 */

const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testView89Bug() {
  const browser = await chromium.launch({ 
    headless: false,
    slowMo: 200 // Slow enough to see the corruption happen
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    recordVideo: {
      dir: 'test/e2e/videos',
      size: { width: 1280, height: 800 }
    }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🐛 Testing View 8/9 Switching Bug\n');
    console.log('Connecting to ttyd at', TTYD_URL);
    
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(2000);
    
    // Take initial screenshot
    await page.screenshot({ path: 'test/e2e/screenshots/bug-initial.png' });
    console.log('📸 Initial state captured');
    
    // First, go to Health view (8)
    console.log('\nNavigating to Health view (8)...');
    await page.keyboard.press('8');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/screenshots/bug-health-first.png' });
    
    // Then Performance view (9)
    console.log('Navigating to Performance view (9)...');
    await page.keyboard.press('9');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/screenshots/bug-performance-first.png' });
    
    // Now rapidly switch between them
    console.log('\n🔄 Rapidly switching between views 8 and 9...');
    const switches = 10;
    
    for (let i = 0; i < switches; i++) {
      console.log(`  Switch ${i + 1}/${switches}: 8 -> 9`);
      
      // Switch to Health (8)
      await page.keyboard.press('8');
      await sleep(300);
      
      // Switch to Performance (9)
      await page.keyboard.press('9');
      await sleep(300);
    }
    
    // Take screenshot of corrupted state
    await page.screenshot({ path: 'test/e2e/screenshots/bug-corrupted.png' });
    console.log('\n📸 Corrupted state captured');
    
    // Try to recover by going to another view
    console.log('\nAttempting recovery by switching to Jobs view (1)...');
    await page.keyboard.press('1');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/screenshots/bug-recovery-attempt.png' });
    
    // Go back to see if still corrupted
    console.log('Going back to Performance view (9)...');
    await page.keyboard.press('9');
    await sleep(1000);
    await page.screenshot({ path: 'test/e2e/screenshots/bug-after-recovery.png' });
    
    console.log('\n✅ Bug reproduction test completed');
    console.log('\n📊 Results:');
    console.log('  - Check screenshots/bug-corrupted.png for the corrupted view');
    console.log('  - Check videos/view-8-9-bug.webm for full recording');
    
  } catch (error) {
    console.error('\n❌ Test failed:', error);
    await page.screenshot({ path: 'test/e2e/screenshots/bug-error.png' });
    throw error;
  } finally {
    // Save video
    const video = page.video();
    if (video) {
      await video.saveAs('test/e2e/videos/view-8-9-bug.webm');
      console.log('\n📹 Video saved: test/e2e/videos/view-8-9-bug.webm');
    }
    
    await context.close();
    await browser.close();
  }
}

// Create directories if needed
const fs = require('fs');
const dirs = ['test/e2e/screenshots', 'test/e2e/videos'];
dirs.forEach(dir => {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
});

// Run the test
testView89Bug().catch(console.error);