#!/usr/bin/env node
// Tests for safety-guard.js hook
// Run: node tests/safety-guard.test.js

const { execFileSync } = require('child_process');
const path = require('path');

const GUARD = path.join(__dirname, '..', 'hooks', 'safety-guard.js');

let passed = 0;
let failed = 0;

function test(name, command, expectedDecision) {
  const input = JSON.stringify({
    tool_name: 'Bash',
    tool_input: { command },
    session_id: 'test-session'
  });

  try {
    const result = execFileSync('node', [GUARD], {
      input,
      encoding: 'utf8',
      timeout: 5000
    });

    const output = result.trim();
    let decision = null;

    if (output) {
      try {
        const parsed = JSON.parse(output);
        decision = parsed.decision;
      } catch (e) {
        decision = 'parse_error';
      }
    } else {
      decision = 'fallthrough';
    }

    if (decision === expectedDecision) {
      console.log(`  PASS: ${name}`);
      passed++;
    } else {
      console.log(`  FAIL: ${name}`);
      console.log(`    Expected: ${expectedDecision}`);
      console.log(`    Got:      ${decision}`);
      console.log(`    Command:  ${command}`);
      failed++;
    }
  } catch (e) {
    if (e.status === 0 && expectedDecision === 'fallthrough') {
      console.log(`  PASS: ${name}`);
      passed++;
    } else {
      console.log(`  FAIL: ${name} (process error: ${e.message})`);
      failed++;
    }
  }
}

console.log('\nBlocked commands (dangerous):');
test('rm -rf /',             'rm -rf /',                              'block');
test('rm -rf /etc',          'rm -rf /etc',                           'block');
test('rm -rf /usr',          'rm -rf /usr',                           'block');
test('rm -rf ~',             'rm -rf ~',                              'block');
test('rm -rf $HOME',         'rm -rf $HOME',                          'block');
test('rm -rf .',             'rm -rf .',                               'block');
test('rm -rf ..',            'rm -rf ..',                              'block');
test('sudo rm -rf',          'sudo rm -rf /tmp/foo',                   'block');
test('DROP DATABASE',        'mysql -e "DROP DATABASE production"',     'block');
test('DROP TABLE',           'psql -c "DROP TABLE users"',             'block');
test('TRUNCATE TABLE',       'mysql -e "TRUNCATE TABLE logs"',         'block');
test('DELETE without WHERE', 'psql -c "DELETE FROM users;"',           'block');
test('mkfs',                 'mkfs.ext4 /dev/sda1',                    'block');
test('dd to device',         'dd if=/dev/zero of=/dev/sda bs=1M',      'block');
test('fork bomb',            ':(){ :|:& };:',                          'block');
test('chmod 777 /',          'chmod -R 777 /',                         'block');
test('force push main',      'git push --force origin main',           'block');
test('force push master',    'git push origin master --force',         'block');
test('kubectl delete ns',    'kubectl delete namespace production',     'block');
test('kubectl delete all',   'kubectl delete pods --all',              'block');
test('shutdown',             'shutdown -h now',                         'block');
test('reboot',               'reboot',                                 'block');

console.log('\nApproved commands (safe):');
test('ls',                   'ls -la',                                 'approve');
test('cat',                  'cat README.md',                          'approve');
test('echo',                 'echo hello',                             'approve');
test('git status',           'git status',                             'approve');
test('git log',              'git log --oneline -10',                  'approve');
test('git diff',             'git diff HEAD~1',                        'approve');
test('git add',              'git add .',                               'approve');
test('git commit',           'git commit -m "test"',                   'approve');
test('git push (no force)',  'git push origin feature-branch',         'approve');
test('git fetch',            'git fetch --all',                        'approve');
test('npm test',             'npm test',                               'approve');
test('npm install',          'npm install express',                    'approve');
test('go test',              'go test ./...',                          'approve');
test('make build',           'make build',                             'approve');
test('kubectl get',          'kubectl get pods -n staging',            'approve');
test('kubectl describe',     'kubectl describe pod foo',               'approve');
test('aws sts',              'aws sts get-caller-identity',            'approve');
test('grep',                 'grep -r "TODO" src/',                    'approve');
test('mkdir',                'mkdir -p /tmp/test',                     'approve');
test('curl',                 'curl https://api.example.com',           'approve');
test('python',               'python3 script.py',                      'approve');
test('node',                 'node server.js',                         'approve');
test('jq',                   'jq ".data" file.json',                   'approve');
test('version flag',         'rustc --version',                        'approve');

console.log('\nFallthrough commands (unknown, prompt user):');
test('docker run',           'docker run -it ubuntu bash',             'fallthrough');
test('terraform apply',      'terraform apply',                        'fallthrough');
test('ansible-playbook',     'ansible-playbook deploy.yml',            'fallthrough');
test('sed in-place',         'sed -i "s/foo/bar/g" file.txt',         'fallthrough');
test('chmod (non-root)',     'chmod 755 script.sh',                    'fallthrough');

console.log('\nNon-Bash tools (always approved):');
{
  const input = JSON.stringify({
    tool_name: 'Read',
    tool_input: { file_path: '/etc/passwd' },
    session_id: 'test-session'
  });

  try {
    const result = execFileSync('node', [GUARD], { input, encoding: 'utf8', timeout: 5000 });
    const parsed = JSON.parse(result.trim());
    if (parsed.decision === 'approve') {
      console.log('  PASS: Read tool auto-approved');
      passed++;
    } else {
      console.log('  FAIL: Read tool not approved');
      failed++;
    }
  } catch (e) {
    console.log('  FAIL: Read tool (error)');
    failed++;
  }
}

console.log(`\n${'='.repeat(40)}`);
console.log(`Results: ${passed} passed, ${failed} failed, ${passed + failed} total`);

if (failed > 0) {
  process.exit(1);
}
