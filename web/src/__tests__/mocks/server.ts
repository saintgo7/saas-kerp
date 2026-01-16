import { setupServer } from 'msw/node';
import { handlers } from './handlers';

// Create MSW server with predefined handlers
export const server = setupServer(...handlers);
