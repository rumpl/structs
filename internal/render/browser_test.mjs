// Run with: node --test internal/render/browser_test.mjs
// Set CHROME_BIN if Chrome/Chromium is not installed in a standard location.
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, mkdtempSync, rmSync, writeFileSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const chrome = [process.env.CHROME_BIN, '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/usr/bin/chromium', '/usr/bin/chromium-browser', '/usr/bin/google-chrome'].find(path => path && existsSync(path));
const root = fileURLToPath(new URL('../../', import.meta.url));
let directory;
let report;
let runNumber = 0;

before(() => {
  if (!chrome) return;
  directory = mkdtempSync(join(tmpdir(), 'structs-browser-'));
  const source = join(directory, 'source');
  mkdirSync(source);
  writeFileSync(join(source, 'types.go'), `package sample
import ctx "context"
type Box struct{ A, B int; Ctx ctx.Context }
func (Box) UniqueMethod(
 firstArgumentWithALongDescriptiveName string,
 secondArgumentWithALongDescriptiveName string,
 thirdArgumentWithALongDescriptiveName string,
 fourthArgumentWithALongDescriptiveName string,
) error { return nil }
type Empty struct{}
type Reader interface { Read([]byte) error }
`);
  mkdirSync(join(source, 'vendor'));
  writeFileSync(join(source, 'vendor', 'broken.go'), 'not valid Go');
  report = execFileSync('go', ['run', './cmd/structs', '--format', 'html', source], { cwd: root, encoding: 'utf8' });
});

after(() => { if (directory) rmSync(directory, { recursive: true, force: true }); });

const browserHelpers = `
const check = (condition, message) => { if (!condition) throw new Error(message); };
const search = document.getElementById('search');
const visible = () => [...document.querySelectorAll('article[data-kind]')].filter(el => !el.hidden);
const toc = () => [...document.querySelectorAll('[data-toc-entry]')].filter(el => !el.hidden);
const query = value => { search.value = value; search.dispatchEvent(new Event('input')); };
const filter = value => document.querySelector('[data-filter="' + value + '"]').click();
`;

function checkInBrowser(checks, width = 1440) {
  const page = join(directory, 'case-' + runNumber + '.html');
  const script = '<script>try {' + browserHelpers + checks +
    ';document.body.dataset.testResult="passed";} catch(error) {document.body.dataset.testResult=error.message;}</script>';
  writeFileSync(page, report.replace('</body>', script + '</body>'));
  const output = execFileSync(chrome, ['--headless', '--disable-gpu', '--no-first-run',
    '--no-default-browser-check', '--user-data-dir=' + join(directory, 'profile-' + runNumber++),
    '--window-size=' + width + ',1000', '--dump-dom', pathToFileURL(page).href],
  { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'], timeout: 30000 });
  assert.match(output, /data-test-result="passed"/,
    output.match(/data-test-result="[^"]*"/)?.[0] || 'Browser assertions did not execute');
}

const cases = {
  'search filters fields and methods and clears empty state': `
    query('ctx.context');
    check(visible().length === 1 && visible()[0].id === 'struct-0-0', 'field search');
    query('uniquemethod SAMPLE');
    check(visible().length === 1 && toc().length === 1, 'method/package AND search');
    query('missing');
    check(visible().length === 0 && !document.getElementById('no-results').hidden, 'empty state');
    document.getElementById('clear').click();
    check(visible().length === 3 && document.getElementById('no-results').hidden, 'clear');
  `,
  'type filters update cards and tree': `
    filter('struct');
    check(visible().length === 2 && toc().length === 2, 'struct filter');
    check(document.querySelector('[data-type-section="interface"]').hidden, 'interface section');
    filter('interface');
    check(visible().length === 1 && visible()[0].id === 'interface-0-0' && toc().length === 1, 'interface filter');
    filter('all');
    check(document.getElementById('result-count').textContent === '3 of 3 types', 'result count');
  `,
  'tree search restores collapsed branches': `
    const nodes = [...document.querySelectorAll('[data-tree-node]')];
    document.getElementById('tree-expand').click();
    check(nodes.every(node => node.open), 'expand all');
    document.getElementById('tree-collapse').click();
    check(nodes.every(node => !node.open), 'collapse all');
    query('uniquemethod');
    check(nodes.filter(node => !node.hidden).every(node => node.open), 'matching branches');
    check(nodes[0].querySelector('[data-tree-count]').textContent === '1', 'filtered count');
    query('');
    check(nodes.every(node => !node.open), 'restored expansion state');
  `,
  'deep links reveal filtered type and focus its card': `
    filter('interface');
    query('Reader');
    document.getElementById('tree-collapse').click();
    window.location.hash = 'struct-0-0';
    window.dispatchEvent(new Event('hashchange'));
    const target = document.getElementById('struct-0-0');
    check(!target.hidden && search.value === '', 'reveal target');
    check(document.activeElement === target, 'focus target');
    check(document.querySelector('[aria-current="location"]').getAttribute('href') === '#struct-0-0', 'active link');
    check([...document.querySelectorAll('[data-tree-node]')].every(node => node.open), 'open target branches');
  `,
};

for (const [name, checks] of Object.entries(cases)) {
  test(name, { skip: !chrome }, () => checkInBrowser(checks));
}

for (const width of [1440, 390]) {
  test('single-line signatures and source tree fit ' + width + 'px viewport', { skip: !chrome }, () => {
    checkInBrowser(`
      const signature = document.querySelector('#struct-0-0 code.function-signature');
      check(signature.textContent.startsWith('func UniqueMethod(') && !signature.textContent.includes('(Box)'), 'receiver omitted');
      check(getComputedStyle(signature).whiteSpace === 'nowrap', 'signature does not wrap');
      check(signature.clientHeight <= parseFloat(getComputedStyle(signature).lineHeight) + 1, 'single line of text excluding scrollbar');
      check(signature.scrollWidth > signature.clientWidth && getComputedStyle(signature).overflowX === 'auto', 'signature scrolling');
      if (window.innerWidth >= 1000) check(document.getElementById('toc').getBoundingClientRect().width >= 360, 'sidebar width');
      const name = document.querySelector('.tree-name');
      name.textContent = 'VeryLongName'.repeat(30);
      check(getComputedStyle(name).textOverflow !== 'ellipsis', 'tree name not truncated');
      check(document.documentElement.scrollWidth <= window.innerWidth, 'no page overflow');
    `, width);
  });
}
