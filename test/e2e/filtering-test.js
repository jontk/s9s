const { chromium } = require('playwright');

// Test configuration
const TTYD_URL = process.env.TTYD_URL || 'http://localhost:7681';
const WAIT_TIME = 1500; // milliseconds between actions

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function testFiltering() {
  const browser = await chromium.launch({ 
    headless: false, // Set to true for CI/CD
    slowMo: 100 // Slow down actions to see what's happening
  });
  
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 }
  });
  
  const page = await context.newPage();
  
  try {
    console.log('🔍 Testing filtering functionality across all views...');
    await page.goto(TTYD_URL);
    await page.waitForLoadState('networkidle');
    await sleep(3000); // Wait for terminal to initialize
    
    // Test views that have filtering capabilities
    const viewsToTest = [
      { key: '1', name: 'Jobs', filter: 'sleep', description: 'Filter jobs containing "sleep"' },
      { key: '2', name: 'Nodes', filter: 'rocky', description: 'Filter nodes containing "rocky"' },
      { key: '3', name: 'Partitions', filter: 'normal', description: 'Filter partitions containing "normal"' },
      { key: '4', name: 'Reservations', filter: 'test', description: 'Filter reservations containing "test"' },
      { key: '5', name: 'QoS', filter: 'normal', description: 'Filter QoS containing "normal"' },
      { key: '6', name: 'Accounts', filter: 'root', description: 'Filter accounts containing "root"' },
      { key: '7', name: 'Users', filter: 'user', description: 'Filter users containing "user"' }
    ];
    
    console.log(`🎯 Testing filtering in ${viewsToTest.length} views...`);
    
    for (const view of viewsToTest) {
      console.log(`\n📊 Testing ${view.name} view filtering...`);
      
      // Switch to view
      await page.keyboard.press(view.key);
      await sleep(WAIT_TIME);
      
      // Take screenshot before filtering
      await page.screenshot({ 
        path: `test/e2e/filter-${view.name.toLowerCase()}-before.png`, 
        fullPage: true 
      });
      
      // Open filter (/)
      console.log(`   - Opening filter in ${view.name} view...`);
      await page.keyboard.press('/');
      await sleep(500);
      
      // Type filter text
      console.log(`   - ${view.description}...`);
      await page.keyboard.type(view.filter);
      await sleep(500);
      
      // Take screenshot while filtering
      await page.screenshot({ 
        path: `test/e2e/filter-${view.name.toLowerCase()}-typing.png`, 
        fullPage: true 
      });
      
      // Apply filter (Enter)
      await page.keyboard.press('Enter');
      await sleep(WAIT_TIME);
      
      // Take screenshot after filtering
      await page.screenshot({ 
        path: `test/e2e/filter-${view.name.toLowerCase()}-after.png`, 
        fullPage: true 
      });
      
      // Clear filter
      console.log(`   - Clearing filter in ${view.name} view...`);
      await page.keyboard.press('/');
      await sleep(500);
      await page.keyboard.press('Control+U'); // Clear line
      await sleep(300);
      await page.keyboard.press('Enter');
      await sleep(WAIT_TIME);
      
      // Take screenshot after clearing filter
      await page.screenshot({ 
        path: `test/e2e/filter-${view.name.toLowerCase()}-cleared.png`, 
        fullPage: true 
      });
    }
    
    // Detailed Jobs view filter testing (since it has the updated layout)
    console.log('\n🎯 Detailed Jobs view filter testing...');
    await page.keyboard.press('1');
    await sleep(WAIT_TIME);
    
    // Test multiple filter scenarios for Jobs
    const jobFilters = ['sleep', 'pending', 'normal', '175'];
    
    for (const filter of jobFilters) {
      console.log(`   - Testing Jobs filter: "${filter}"...`);
      
      // Apply filter
      await page.keyboard.press('/');
      await sleep(300);
      await page.keyboard.type(filter);
      await sleep(300);
      await page.screenshot({ 
        path: `test/e2e/jobs-filter-${filter}.png`, 
        fullPage: true 
      });
      await page.keyboard.press('Enter');
      await sleep(WAIT_TIME);
      
      // Screenshot results
      await page.screenshot({ 
        path: `test/e2e/jobs-filter-${filter}-results.png`, 
        fullPage: true 
      });
      
      // Clear for next test
      await page.keyboard.press('/');
      await sleep(300);
      await page.keyboard.press('Control+U');
      await sleep(300);
      await page.keyboard.press('Enter');
      await sleep(WAIT_TIME);
    }
    
    // Test Jobs multi-select + filtering
    console.log('   - Testing multi-select mode with filtering...');
    await page.keyboard.press('v'); // Enable multi-select
    await sleep(WAIT_TIME);
    
    // Apply filter in multi-select mode
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.type('sleep');
    await sleep(300);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    
    // Try to select filtered rows
    await page.keyboard.press('ArrowDown');
    await sleep(300);
    await page.keyboard.press(' '); // Toggle selection
    await sleep(300);
    
    await page.screenshot({ 
      path: 'test/e2e/jobs-multiselect-with-filter.png', 
      fullPage: true 
    });
    
    // Clear and disable multi-select
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.press('Control+U');
    await sleep(300);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    await page.keyboard.press('v'); // Disable multi-select
    await sleep(WAIT_TIME);
    
    // Test filter edge cases
    console.log('   - Testing filter edge cases...');
    
    // Empty filter
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.press('Enter'); // Empty filter should show all
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/filter-empty-test.png', 
      fullPage: true 
    });
    
    // Non-matching filter
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.type('nonexistentjob123');
    await sleep(300);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/filter-no-matches.png', 
      fullPage: true 
    });
    
    // Clear final filter
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.press('Control+U');
    await sleep(300);
    await page.keyboard.press('Enter');
    await sleep(WAIT_TIME);
    
    // Test ESC to cancel filter input
    console.log('   - Testing ESC to cancel filter input...');
    await page.keyboard.press('/');
    await sleep(300);
    await page.keyboard.type('testcancel');
    await sleep(300);
    await page.keyboard.press('Escape'); // Should cancel without applying
    await sleep(WAIT_TIME);
    await page.screenshot({ 
      path: 'test/e2e/filter-esc-cancel.png', 
      fullPage: true 
    });
    
    console.log('✅ Filtering test completed successfully!');
    console.log('📸 Screenshots saved showing filter functionality across all views');
    console.log('🔍 Tested:');
    console.log('   - Basic filtering in Jobs, Nodes, Partitions, Reservations, QoS, Accounts, Users views');
    console.log('   - Multiple filter terms in Jobs view');
    console.log('   - Multi-select mode + filtering');  
    console.log('   - Edge cases: empty filter, no matches, ESC cancel');
    console.log('   - Filter layout compatibility with updated Jobs view');
    
  } catch (error) {
    console.error('❌ Filtering test failed:', error);
    await page.screenshot({ path: 'test/e2e/filtering-test-error.png', fullPage: true });
    throw error;
  } finally {
    await browser.close();
  }
}

// Run the test
testFiltering().catch(console.error);