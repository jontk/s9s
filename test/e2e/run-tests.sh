#!/usr/bin/env sh

echo "🚀 Installing Playwright dependencies..."
cd test/e2e
npm install

echo "\n📁 Creating test directories..."
mkdir -p screenshots videos

echo "\n🧪 Running s9s TUI tests..."
npm run test:advanced

echo "\n✅ Tests complete! Check the following:"
echo "   - Screenshots: test/e2e/screenshots/"
echo "   - Videos: test/e2e/videos/"