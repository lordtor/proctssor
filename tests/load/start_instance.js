import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const instanceStartRate = new Rate('instance_start_rate');
const instanceStartDuration = new Trend('instance_start_duration');
const dbConnectionPoolUsage = new Gauge('db_connection_pool_usage');
const errorRate = new Rate('error_rate');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_VERSION = '/api/v1';

// Default test process ID
const DEFAULT_PROCESS_ID = __ENV.PROCESS_ID || 'test-process';

// Options for different scenarios
export const options = {
  scenarios: {
    // Smoke test - быстрая проверка работоспособности
    smoke: {
      executor: 'constant-vus',
      vus: 10,
      duration: '30s',
      tags: { scenario: 'smoke' },
    },
    // Load test - нагрузочное тестирование
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },  // ramp up
        { duration: '5m', target: 100 },  // sustain
        { duration: '2m', target: 200 },  // ramp up
        { duration: '5m', target: 200 },  // sustain
        { duration: '2m', target: 0 },    // ramp down
      ],
      tags: { scenario: 'load' },
    },
    // Stress test - стресс-тестирование до отказа
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 100 },
        { duration: '2m', target: 200 },
        { duration: '2m', target: 300 },
        { duration: '2m', target: 400 },
        { duration: '2m', target: 500 },
        { duration: '5m', target: 0 },
      ],
      tags: { scenario: 'stress' },
    },
    // Spike test - резкий скачок нагрузки
    spike: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },
        { duration: '10s', target: 500 },
        { duration: '5m', target: 500 },
        { duration: '30s', target: 0 },
      ],
      tags: { scenario: 'spike' },
    },
  },
  // Thresholds
  thresholds: {
    'http_req_duration': ['p(50) < 200', 'p(95) < 500', 'p(99) < 1000'],
    'http_req_failed': ['rate < 0.001'], // error rate < 0.1%
    'instance_start_duration': ['p(95) < 500'],
    'instance_start_rate': ['rate > 0.99'], // 99% success rate
  },
};

// Helper function to generate random data
function randomString(length) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

// Setup function - runs once per VU
export function setup() {
  // Deploy test process if it doesn't exist
  const deployPayload = {
    id: DEFAULT_PROCESS_ID,
    key: DEFAULT_PROCESS_ID,
    name: 'Load Test Process',
    bpmnXml: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="${DEFAULT_PROCESS_ID}" name="Load Test Process" isExecutable="true">
    <startEvent id="start" name="Start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="userTask"/>
    <userTask id="userTask" name="Review"/>
    <sequenceFlow id="flow2" sourceRef="userTask" targetRef="end"/>
    <endEvent id="end" name="End"/>
  </process>
</definitions>`,
  };

  const deployUrl = `${BASE_URL}${API_VERSION}/processes`;
  const deployRes = http.post(deployUrl, JSON.stringify(deployPayload), {
    headers: {
      'Content-Type': 'application/json',
    },
  });

  check(deployRes, {
    'process deployed': (r) => r.status === 201 || r.status === 200 || r.status === 409, // 409 if already exists
  });

  return {
    processId: DEFAULT_PROCESS_ID,
  };
}

// Main test function
export default function(data) {
  const processId = data.processId;

  // Test 1: Start instance
  startInstance(processId);

  // Test 2: List instances (with pagination)
  listInstances();

  // Test 3: Get instance by ID (if we have one)
  // This would require storing instance IDs from startInstance

  // Small delay between requests
  sleep(0.1);
}

// Start a new process instance
function startInstance(processId) {
  const url = `${BASE_URL}${API_VERSION}/processes/${processId}/start`;
  const payload = {
    variables: {
      projectName: `test-project-${randomString(8)}`,
      initiator: `user-${__VU}`,
      timestamp: new Date().toISOString(),
    },
    initiator: `load-test-user-${__VU}`,
  };

  const startTime = Date.now();
  const res = http.post(url, JSON.stringify(payload), {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: {
      operation: 'start_instance',
    },
  });
  const duration = Date.now() - startTime;

  // Record custom metrics
  instanceStartDuration.add(duration);

  const success = check(res, {
    'start instance status is 201': (r) => r.status === 201,
    'start instance has instance_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.instance_id !== undefined && body.instance_id !== '';
      } catch (e) {
        return false;
      }
    },
    'start instance duration < 500ms': () => duration < 500,
  });

  instanceStartRate.add(success);
  errorRate.add(!success);

  // Try to parse and store instance ID for later tests
  if (success) {
    try {
      const body = JSON.parse(res.body);
      if (body.instance_id) {
        // Store for potential use in other tests
        // Note: In k6, we can't easily share data between iterations
      }
    } catch (e) {
      // Ignore parsing errors
    }
  }

  return res;
}

// List instances with filters
function listInstances() {
  const url = `${BASE_URL}${API_VERSION}/instances?limit=10&status=active`;

  const res = http.get(url, {
    tags: {
      operation: 'list_instances',
    },
  });

  const success = check(res, {
    'list instances status is 200': (r) => r.status === 200,
    'list instances returns array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!success);

  return res;
}

// Get instance details
export function getInstance(instanceId) {
  const url = `${BASE_URL}${API_VERSION}/instances/${instanceId}`;

  const res = http.get(url, {
    tags: {
      operation: 'get_instance',
    },
  });

  check(res, {
    'get instance status is 200': (r) => r.status === 200,
    'get instance has id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  return res;
}

// Get instance variables
export function getVariables(instanceId) {
  const url = `${BASE_URL}${API_VERSION}/instances/${instanceId}/variables`;

  const res = http.get(url, {
    tags: {
      operation: 'get_variables',
    },
  });

  check(res, {
    'get variables status is 200': (r) => r.status === 200,
  });

  return res;
}

// Complete user task
export function completeTask(instanceId, taskId) {
  const url = `${BASE_URL}${API_VERSION}/instances/${instanceId}/tasks/${taskId}/complete`;
  const payload = {
    variables: {
      approved: true,
      comment: 'Completed via load test',
    },
    user_id: `load-test-user-${__VU}`,
  };

  const res = http.post(url, JSON.stringify(payload), {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: {
      operation: 'complete_task',
    },
  });

  check(res, {
    'complete task status is 200': (r) => r.status === 200,
  });

  return res;
}

// Teardown function - runs once at the end
export function teardown(data) {
  console.log('Load test completed');
  console.log(`Process ID used: ${data.processId}`);
}

// Handle summary - generate custom report
export function handleSummary(data) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    [`reports/load-test-${timestamp}.json`]: JSON.stringify(data, null, 2),
    [`reports/load-test-summary-${timestamp}.html`]: htmlReport(data),
  };
}

// Generate HTML report
function htmlReport(data) {
  return `
<!DOCTYPE html>
<html>
<head>
  <title>k6 Load Test Report</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; }
    h1 { color: #333; }
    table { border-collapse: collapse; width: 100%; margin: 20px 0; }
    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background-color: #4CAF50; color: white; }
    tr:nth-child(even) { background-color: #f2f2f2; }
    .metric { font-weight: bold; }
    .passed { color: green; }
    .failed { color: red; }
    .thresholds { margin: 20px 0; }
  </style>
</head>
<body>
  <h1>k6 Load Test Report</h1>
  <p>Generated: ${new Date().toISOString()}</p>
  
  <h2>Test Configuration</h2>
  <table>
    <tr><th>Parameter</th><th>Value</th></tr>
    <tr><td>Base URL</td><td>${BASE_URL}</td></tr>
    <tr><td>Process ID</td><td>${DEFAULT_PROCESS_ID}</td></tr>
  </table>
  
  <h2>Metrics Summary</h2>
  <table>
    <tr><th>Metric</th><th>Average</th><th>Min</th><th>Max</th><th>p(50)</th><th>p(95)</th><th>p(99)</th></tr>
    <tr>
      <td class="metric">HTTP Request Duration</td>
      <td>${data.metrics.http_req_duration?.avg?.toFixed(2) || 'N/A'}ms</td>
      <td>${data.metrics.http_req_duration?.min?.toFixed(2) || 'N/A'}ms</td>
      <td>${data.metrics.http_req_duration?.max?.toFixed(2) || 'N/A'}ms</td>
      <td>${data.metrics.http_req_duration?.med?.toFixed(2) || 'N/A'}ms</td>
      <td>${data.metrics.http_req_duration?.['p(95)']?.toFixed(2) || 'N/A'}ms</td>
      <td>${data.metrics.http_req_duration?.['p(99)']?.toFixed(2) || 'N/A'}ms</td>
    </tr>
    <tr>
      <td class="metric">HTTP Requests</td>
      <td colspan="6">${data.metrics.http_reqs?.count || 0} total</td>
    </tr>
    <tr>
      <td class="metric">Error Rate</td>
      <td colspan="6">${((data.metrics.http_req_failed?.rate || 0) * 100).toFixed(2)}%</td>
    </tr>
  </table>
  
  <h2>Thresholds</h2>
  <div class="thresholds">
    ${Object.entries(data.thresholds || {}).map(([name, result]) => `
      <p class="${result ? 'passed' : 'failed'}">
        ${result ? '✓' : '✗'} ${name}
      </p>
    `).join('')}
  </div>
  
  <h2>Checks</h2>
  <table>
    <tr><th>Check</th><th>Passes</th><th>Failures</th><th>Pass Rate</th></tr>
    ${Object.entries(data.checks || {}).map(([name, check]) => `
      <tr>
        <td>${name}</td>
        <td>${check.passes}</td>
        <td>${check.fails}</td>
        <td class="${check.fails === 0 ? 'passed' : 'failed'}">
          ${((check.passes / (check.passes + check.fails)) * 100).toFixed(2)}%
        </td>
      </tr>
    `).join('')}
  </table>
</body>
</html>
  `;
}

// Text summary for console
function textSummary(data, options) {
  // Simple text summary
  return `
k6 Load Test Summary
====================

Test Duration: ${data.state?.testRunDurationMs ? (data.state.testRunDurationMs / 1000).toFixed(2) : 'N/A'}s

HTTP Requests:
  - Total: ${data.metrics.http_reqs?.count || 0}
  - Failed: ${data.metrics.http_req_failed?.count || 0}
  - Error Rate: ${((data.metrics.http_req_failed?.rate || 0) * 100).toFixed(2)}%

Latency (ms):
  - Average: ${data.metrics.http_req_duration?.avg?.toFixed(2) || 'N/A'}
  - p50: ${data.metrics.http_req_duration?.med?.toFixed(2) || 'N/A'}
  - p95: ${data.metrics.http_req_duration?.['p(95)']?.toFixed(2) || 'N/A'}
  - p99: ${data.metrics.http_req_duration?.['p(99)']?.toFixed(2) || 'N/A'}

Threshold Results:
${Object.entries(data.thresholds || {}).map(([name, result]) => `  ${result ? '✓' : '✗'} ${name}`).join('\n')}
`;
}
