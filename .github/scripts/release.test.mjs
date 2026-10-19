// These tests stay offline; GitHub and GHCR access lives in the workflow commands.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  semver,
  compare,
  aliases,
  aliasAction,
  verifyIdentity,
  expectedAssets,
  assetPlan,
} from './release.mjs';

test('strict release SemVer, including numeric prerelease identifiers', () => {
  for (const v of ['0.1.0', '1.0.0-rc.1', '1.0.0-rc.10']) assert.doesNotThrow(() => semver(v));
  for (const v of ['v1.0.0', '01.0.0', '1.0', '1.0.0-rc.01', '1.0.0;echo bad', '1.0.0\n'])
    assert.throws(() => semver(v));
  assert.equal(compare('1.0.0-rc.10', '1.0.0-rc.2'), 1);
  assert.equal(compare('1.0.0', '1.0.0-rc.10'), 1);
  assert.equal(compare('0.2.9', '0.3.0'), -1);
  assert.equal(compare('2.0.0', '1.9.9'), 1);
});

test('Docker aliases exclude pre-1.0 major and prereleases', () => {
  assert.deepEqual(aliases('0.2.3'), ['0.2', 'latest']);
  assert.deepEqual(aliases('1.2.3'), ['1.2', '1', 'latest']);
  assert.deepEqual(aliases('1.0.0-rc.1'), []);
});

// A consistent release baseline; each patch below breaks one source-of-truth check.
const identity = {
  tag: 'v0.1.0',
  version: '0.1.0',
  sha: 'a'.repeat(40),
  resolved: 'a'.repeat(40),
  manifest: { '.': '0.1.0' },
  onMain: true,
  release: { tag_name: 'v0.1.0', draft: false, prerelease: false },
};

test('tag, SHA, manifest, main ancestry and release must agree', () => {
  assert.doesNotThrow(() => verifyIdentity(identity));
  for (const patch of [
    { tag: 'v0.2.0' },
    { sha: 'b'.repeat(40) },
    { manifest: { '.': '0.2.0' } },
    { onMain: false },
    { release: null },
    { release: { ...identity.release, draft: true } },
    { release: { ...identity.release, prerelease: true } },
  ]) {
    assert.throws(() => verifyIdentity({ ...identity, ...patch }));
  }
});

test('recovery preserves matching assets, uploads missing and rejects conflicts before uploading', () => {
  assert.equal(expectedAssets('0.1.0').length, 7);
  const local = { archive: 'abc', checksums: 'def' };
  assert.deepEqual(assetPlan(local, {}), ['archive', 'checksums']);
  assert.deepEqual(assetPlan(local, { archive: 'abc' }), ['checksums']);
  assert.deepEqual(assetPlan(local, local), []);
  assert.throws(() => assetPlan(local, { checksums: 'conflict' }), /Conflicting/);
});

test('older recovery preserves newer aliases and retries matching aliases safely', () => {
  const current = { digest: 'sha256:abc' };
  assert.equal(
    aliasAction('0.1.0', current, { version: '0.2.0', digest: 'sha256:new' }),
    'preserve',
  );
  assert.equal(
    aliasAction('0.2.0', current, { version: '0.1.0', digest: 'sha256:old' }),
    'promote',
  );
  assert.equal(aliasAction('0.2.0', current, null), 'promote');
  assert.equal(aliasAction('0.2.0', current, { version: '0.2.0', digest: 'sha256:abc' }), 'verify');
  assert.throws(
    () => aliasAction('0.2.0', current, { version: '0.2.0', digest: 'sha256:other' }),
    /Conflicting/,
  );
});
