// Exercise the actual Release Please strategy used by the pinned action, offline.
const assert = require('node:assert/strict');
const {createRequire} = require('node:module');
const path = require('node:path');
// CI installs these tools in a temporary prefix to keep npm dependencies out of the repo.
const tools = createRequire(path.join(process.env.RELEASE_TOOLS, 'package.json'));
tools('release-please'); // Load the factory first to initialize its circular imports.
const {Go} = tools('release-please/build/src/strategies/go');
const {Version} = tools('release-please/build/src/version');
const {TagName} = tools('release-please/build/src/util/tag-name');
const {parseConventionalCommits} = tools('release-please/build/src/commit');
const {buildVersioningStrategy} = tools('release-please/build/src/factories/versioning-strategy-factory');
const config = require('../../release-please-config.json').packages['.'];
const github = {repository: {owner: 'tomasweigenast', repo: 'pdfgen'}};
const logger = {info() {}, debug() {}, warn() {}, error() {}};

// Build a release PR in memory: no API calls, tags or real repository writes.
// previous=null models the empty manifest used on the first release.
async function candidate(message, previous = '0.1.0', extra = {}) {
  const options = {github, targetBranch: 'main', logger,
    includeComponentInTag: false, initialVersion: config['initial-version'],
    changelogSections: config['changelog-sections'],
    bumpMinorPreMajor: config['bump-minor-pre-major'],
    bumpPatchForMinorPreMajor: config['bump-patch-for-minor-pre-major'], ...extra};
  options.versioningStrategy = buildVersioningStrategy({...options, type: extra.versioning});
  const strategy = new Go(options);
  const commits = parseConventionalCommits([{sha: 'a'.repeat(40), message, files: ['cmd/pdfgen/main.go']}], logger);
  const latest = previous ? {tag: new TagName(Version.parse(previous), undefined, undefined, true), sha: 'b'.repeat(40)} : undefined;
  return strategy.buildReleasePullRequest(commits, latest);
}
(async () => {
  assert.equal((await candidate('feat: ship releases', null)).version.toString(), '0.1.0');
  for (const [message, version] of [['feat: add rendering', '0.2.0'], ['fix: correct output', '0.1.1'],
    ['perf: reduce allocations', '0.1.1'], ['feat!: replace API', '0.2.0'],
    ['refactor!: replace API', '0.2.0'], ['docs: explain API\n\nBREAKING CHANGE: remove old API', '0.2.0']]) {
    const pr = await candidate(message);
    assert.equal(pr.version.toString(), version, message);
    if (message.includes('!') || message.includes('BREAKING CHANGE')) assert.match(pr.body.toString(), /BREAKING CHANGES/);
  }
  // Hidden changelog types must also produce no release on initial onboarding.
  for (const type of ['docs', 'chore', 'chore(deps)', 'refactor', 'test', 'build', 'ci', 'deps']) {
    assert.equal(await candidate(`${type}: maintenance`), undefined, type);
    assert.equal(await candidate(`${type}: maintenance`, null), undefined, `initial ${type}`);
  }
  // Check explicit graduation, RC entry/numbering and final promotion separately.
  assert.equal((await candidate('fix: prepare stable\n\nRelease-As: 1.0.0')).version.toString(), '1.0.0');
  assert.equal((await candidate('feat!: change API', '1.0.0')).version.toString(), '2.0.0');
  assert.equal((await candidate('feat: prepare RC\n\nRelease-As: 1.0.0-rc.1', '0.1.0',
    {versioning: 'prerelease', prereleaseType: 'rc', prerelease: true})).version.toString(), '1.0.0-rc.1');
  assert.equal((await candidate('fix: refine RC', '1.0.0-rc.1',
    {versioning: 'prerelease', prereleaseType: 'rc', prerelease: true})).version.toString(), '1.0.0-rc.2');
  assert.equal((await candidate('fix: promote stable\n\nRelease-As: 1.0.0', '1.0.0-rc.2')).version.toString(), '1.0.0');
  console.log('Release Please policy fixtures passed');
})().catch(error => { console.error(error); process.exitCode = 1; });
