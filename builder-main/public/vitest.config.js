import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['js/**/*.test.js'],
    // Increase timeout for DOM tests
    testTimeout: 10000,
  },
});
