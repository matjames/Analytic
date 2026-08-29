import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const clientSource = readFileSync(path.join(root, 'src/api/client.ts'), 'utf8');
const handlerSource = readFileSync(path.join(root, '../backend/pkg/api/handlers.go'), 'utf8');

const methodNames = {
  MethodDelete: 'DELETE',
  MethodGet: 'GET',
  MethodHead: 'HEAD',
  MethodPost: 'POST',
  MethodPut: 'PUT',
};

const backendRoutes = [];
const routePattern = /router\.HandleFunc\("([^"]+)"[^\n]*?\.Methods\(([^)]*)\)/g;
for (const match of handlerSource.matchAll(routePattern)) {
  for (const token of match[2].matchAll(/http\.(Method\w+)/g)) {
    const method = methodNames[token[1]];
    if (method) backendRoutes.push({ method, path: match[1] });
  }
}

const clientOperations = [];
const functionPattern = /export async function\s+(\w+)[\s\S]*?(?=\nexport async function|\s*$)/g;
for (const functionMatch of clientSource.matchAll(functionPattern)) {
  const body = functionMatch[0];
  const fetchMatch = body.match(/apiFetch\(\s*`\$\{BASE_URL\}([^`]+)`/);
  if (!fetchMatch) continue;

  const methodMatch = body.match(/method:\s*'([A-Z]+)'/);
  clientOperations.push({
    name: functionMatch[1],
    method: methodMatch?.[1] ?? 'GET',
    path: normalizeClientPath(fetchMatch[1]),
  });
}

function normalizeClientPath(value) {
  return value
    .replace(/\$\{query(?:\.size)?[\s\S]*$/, '')
    .replace(/\$\{suffix\}$/, '')
    .split('?')[0]
    .replace(/\$\{[^}]+\}/g, '{param}')
    .replace(/\/$/, '') || '/';
}

function sameRoute(left, right) {
  const leftParts = left.split('/');
  const rightParts = right.split('/');
  return leftParts.length === rightParts.length && leftParts.every((part, index) => (
    part === rightParts[index] ||
    (/^\{[^}]+\}$/.test(part) && /^\{[^}]+\}$/.test(rightParts[index]))
  ));
}

const missing = clientOperations.filter((operation) => !backendRoutes.some((route) => (
  route.method === operation.method && sameRoute(route.path, operation.path)
)));

if (missing.length > 0) {
  console.error('Frontend API operations without a matching backend route:');
  for (const operation of missing) {
    console.error(`- ${operation.name}: ${operation.method} ${operation.path}`);
  }
  process.exit(1);
}

const coveredRoutes = backendRoutes.filter((route) => clientOperations.some((operation) => (
  route.method === operation.method && sameRoute(route.path, operation.path)
)));
const uncoveredRoutes = backendRoutes.filter((route) => !coveredRoutes.includes(route));

console.log(`Verified ${clientOperations.length} frontend API operations against ${backendRoutes.length} backend routes.`);
console.log(`Backend route coverage: ${coveredRoutes.length}/${backendRoutes.length}.`);
if (uncoveredRoutes.length > 0) {
  console.log('Backend routes without a typed frontend client operation:');
  for (const route of uncoveredRoutes) console.log(`- ${route.method} ${route.path}`);
}
