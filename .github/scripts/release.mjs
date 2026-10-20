// Called by publish.yml to verify a release, recover archives and publish Docker aliases.
// RELEASE_TAG identifies the release; RELEASE_SHA checks its source commit when supplied.
// GH_TOKEN is used by gh and the GitHub/GHCR HTTP requests. Commands below write
// step outputs to GITHUB_OUTPUT so the workflow can decide what to run next.
import { execFileSync } from 'node:child_process';
import { appendFileSync, readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { pathToFileURL } from 'node:url';

const repository = 'tomasweigenast/pdfgen';
const image = `ghcr.io/${repository}`;

// Pass arguments directly to the process; dispatch inputs must never become shell code.
const run = (command, args) => execFileSync(command, args, { encoding: 'utf8' }).trim();
const git = (...args) => run('git', args);
const gh = (...args) => run('gh', args);
const api = (path) => JSON.parse(gh('api', path));

const output = (key, value) => {
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `${key}=${value}\n`);
  console.log(`${key}=${value}`);
};

// Versions are unprefixed here; only Git tags carry the leading v.
export function semver(value) {
  const match =
    /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/.exec(
      value,
    );
  if (!match || match[4]?.split('.').some((p) => /^\d+$/.test(p) && p.length > 1 && p[0] === '0')) {
    throw new Error(`Invalid SemVer: ${value}`);
  }
  return { numbers: match.slice(1, 4).map(BigInt), pre: match[4]?.split('.') ?? [] };
}

// SemVer precedence: a stable version follows its RCs, and rc.10 follows rc.2.
export function compare(a, b) {
  a = semver(a);
  b = semver(b);

  for (let i = 0; i < 3; i++) {
    if (a.numbers[i] !== b.numbers[i]) return a.numbers[i] > b.numbers[i] ? 1 : -1;
  }
  if (!a.pre.length || !b.pre.length) return Number(!a.pre.length) - Number(!b.pre.length);
  for (let i = 0; i < Math.max(a.pre.length, b.pre.length); i++) {
    if (a.pre[i] === b.pre[i]) continue;
    if (a.pre[i] === undefined) return -1;
    if (b.pre[i] === undefined) return 1;
    const an = /^\d+$/.test(a.pre[i]),
      bn = /^\d+$/.test(b.pre[i]);
    if (an && bn) return BigInt(a.pre[i]) > BigInt(b.pre[i]) ? 1 : -1;
    if (an !== bn) return an ? -1 : 1;
    return a.pre[i] > b.pre[i] ? 1 : -1;
  }
  return 0;
}

// RCs get their exact version tag only. Before 1.0, do not publish a moving "0" tag.
export function aliases(version) {
  const {
    numbers: [major, minor],
    pre,
  } = semver(version);
  if (pre.length) return [];
  return [`${major}.${minor}`, ...(major > 0n ? [`${major}`] : []), 'latest'];
}

// All checks use the manifest at the tagged commit, which may be older than main.
export function verifyIdentity({ tag, version, sha, resolved, manifest, onMain, release }) {
  semver(version);
  if (tag !== `v${version}`) throw new Error('Tag/version mismatch');
  if (!/^[0-9a-f]{40}$/.test(sha) || resolved !== sha) throw new Error('SHA mismatch');
  if (manifest['.'] !== version) throw new Error('Manifest version mismatch');
  if (!onMain) throw new Error('Release commit is not on main');
  if (!release || release.draft || release.tag_name !== tag)
    throw new Error('Missing published release');
  if (release.prerelease !== Boolean(semver(version).pre.length))
    throw new Error('Prerelease classification mismatch');
}

// Retrying an old release must not move latest or a major/minor tag backwards.
export function aliasAction(version, current, existing) {
  if (existing && compare(existing.version, version) > 0) return 'preserve';
  if (existing?.version === version && existing.digest !== current.digest)
    throw new Error('Conflicting alias');
  return existing?.digest === current.digest ? 'verify' : 'promote';
}

export function expectedAssets(version) {
  semver(version);
  return ['darwin', 'linux', 'windows']
    .flatMap((os) =>
      ['amd64', 'arm64'].map(
        (arch) => `pdfgen_${version}_${os}_${arch}.${os === 'windows' ? 'zip' : 'tar.gz'}`,
      ),
    )
    .concat(`pdfgen_${version}_checksums.txt`);
}

// Compute the whole upload list first, so a conflict stops us before any upload.
// local and remote map filenames to SHA-256 hashes.
export function assetPlan(local, remote) {
  const uploads = [];

  for (const [name, hash] of Object.entries(local)) {
    if (remote[name] && remote[name] !== hash) throw new Error(`Conflicting asset: ${name}`);
    if (!remote[name]) uploads.push(name);
  }
  return uploads;
}
const digest = (bytes) => createHash('sha256').update(bytes).digest('hex');

async function verify() {
  const tag = process.env.RELEASE_TAG;
  if (!tag?.startsWith('v')) throw new Error('Expected a v-prefixed release tag');
  const version = tag.slice(1);
  semver(version);
  // ^{commit} also resolves annotated tags to the commit they point at.
  const resolved = git('rev-parse', '--verify', `refs/tags/${tag}^{commit}`);
  const sha = process.env.RELEASE_SHA || resolved;
  let onMain = true;
  try {
    git('merge-base', '--is-ancestor', sha, 'origin/main');
  } catch {
    onMain = false;
  }
  const release = api(`repos/${repository}/releases/tags/${tag}`);
  verifyIdentity({
    tag,
    version,
    sha,
    resolved,
    manifest: JSON.parse(git('show', `${sha}:.release-please-manifest.json`)),
    onMain,
    release,
  });
  output('sha', sha);
  output('tag', tag);
  output('version', version);
  // The caller may have tested a different main commit than the one tagged.
  output('needs_ci', String(sha !== process.env.VALIDATED_SHA));
}

// RECOVERY_RUN_ID opts into reusing a previous run's exact dist files. Without
// it, the workflow rebuilds from the release commit using the pinned toolchain.
async function restore() {
  const runId = process.env.RECOVERY_RUN_ID;
  if (!runId) {
    output('restored', 'false');
    return;
  }
  if (!/^[1-9]\d*$/.test(runId)) throw new Error('Invalid recovery run ID');
  const tag = process.env.RELEASE_TAG;
  semver(tag.slice(1));
  const artifacts = JSON.parse(
    gh(
      'api',
      '--paginate',
      '--slurp',
      `repos/${repository}/actions/runs/${runId}/artifacts?per_page=100`,
    ),
  )
    .flatMap((page) => page.artifacts)
    .filter((a) => !a.expired && a.name.startsWith(`release-${tag}-`));
  // Rerunning jobs can leave several artifacts; use the newest matching attempt.
  artifacts.sort((a, b) => b.id - a.id);
  if (!artifacts.length) throw new Error('No retained release artifact found in that run');
  gh('run', 'download', runId, '--repo', repository, '--name', artifacts[0].name, '--dir', 'dist');
  for (const name of expectedAssets(tag.slice(1))) readFileSync(`dist/${name}`);
  output('restored', 'true');
}

async function assets(checkOnly = false) {
  const tag = process.env.RELEASE_TAG;
  const version = tag.slice(1);
  const names = expectedAssets(version);
  const local = Object.fromEntries(
    names.map((name) => [name, digest(readFileSync(`dist/${name}`))]),
  );
  // A failed build may leave an incomplete dist directory. Check every archive
  // against the generated checksum file before inspecting or uploading assets.
  const sums = new Map(
    readFileSync(`dist/pdfgen_${version}_checksums.txt`, 'utf8')
      .trim()
      .split('\n')
      .map((line) => {
        const match = /^([a-f0-9]{64})  (\S+)$/.exec(line);
        if (!match) throw new Error('Malformed checksum file');
        return [match[2], match[1]];
      }),
  );
  for (const name of names.slice(0, -1))
    if (sums.get(name) !== local[name]) throw new Error(`Invalid checksum: ${name}`);
  const release = api(`repos/${repository}/releases/tags/${tag}`);
  const remote = {};
  // Compare the actual published bytes, including the checksum file itself.
  // Matching assets are left alone; a different file with the same name is an error.
  const published = JSON.parse(
    gh(
      'api',
      '--paginate',
      '--slurp',
      `repos/${repository}/releases/${release.id}/assets?per_page=100`,
    ),
  ).flat();
  for (const asset of published.filter((a) => names.includes(a.name))) {
    const response = await fetch(
      `https://api.github.com/repos/${repository}/releases/assets/${asset.id}`,
      {
        headers: {
          Authorization: `Bearer ${process.env.GH_TOKEN}`,
          Accept: 'application/octet-stream',
        },
      },
    );
    if (!response.ok) throw new Error(`Cannot read ${asset.name}: ${response.status}`);
    remote[asset.name] = digest(Buffer.from(await response.arrayBuffer()));
  }
  // check-assets runs this same comparison after upload, but cannot mutate the release.
  const missing = assetPlan(local, remote);
  if (checkOnly && missing.length) throw new Error(`Missing assets: ${missing.join(', ')}`);
  for (const name of checkOnly ? [] : missing)
    gh('release', 'upload', tag, `dist/${name}`, '--repo', repository);
  output('assets', missing.length ? `Uploaded ${missing.length}` : 'All assets match');
}

let registryToken;
async function registry(path, optional = false) {
  if (!registryToken) {
    // GHCR exchanges the workflow credentials for a registry-scoped bearer token.
    const auth = Buffer.from(`${process.env.GITHUB_ACTOR}:${process.env.GH_TOKEN}`).toString(
      'base64',
    );
    const res = await fetch(
      `https://ghcr.io/token?service=ghcr.io&scope=repository:${repository}:pull`,
      {
        headers: { Authorization: `Basic ${auth}` },
      },
    );
    if (!res.ok) throw new Error(`GHCR authentication: ${res.status}`);
    registryToken = (await res.json()).token;
  }
  const res = await fetch(`https://ghcr.io/v2/${repository}/${path}`, {
    headers: {
      Authorization: `Bearer ${registryToken}`,
      Accept:
        'application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json',
    },
  });
  // Only a missing manifest permits a new build. Auth/network failures must stop it.
  if (optional && res.status === 404) return null;
  if (!res.ok) throw new Error(`GHCR ${path}: ${res.status}`);
  return { body: await res.json(), digest: res.headers.get('docker-content-digest') };
}

// Verify both platform manifests, not just the labels of whichever image is local.
async function inspect(ref) {
  const index = await registry(`manifests/${ref}`, true);
  if (!index) return null;
  // BuildKit also adds attestation manifests with unknown/unknown platforms.
  const platforms = (index.body.manifests ?? []).filter((m) => m.platform?.os === 'linux');
  if (
    platforms.length !== 2 ||
    !['amd64', 'arm64'].every((arch) => platforms.some((m) => m.platform.architecture === arch))
  ) {
    throw new Error(`Unexpected platforms in ${ref}`);
  }
  const labels = [];
  for (const platform of platforms) {
    const manifest = await registry(`manifests/${platform.digest}`);
    const config = await registry(`blobs/${manifest.body.config.digest}`);
    labels.push(config.body.config.Labels);
  }
  if (
    labels.some(
      (l) => !l || l['org.opencontainers.image.source'] !== `https://github.com/${repository}`,
    )
  ) {
    throw new Error(`Invalid image source: ${ref}`);
  }
  for (const key of ['org.opencontainers.image.version', 'org.opencontainers.image.revision']) {
    if (labels[0][key] !== labels[1][key]) throw new Error(`Inconsistent ${key}: ${ref}`);
  }
  semver(labels[0]['org.opencontainers.image.version']);
  return {
    digest: index.digest,
    version: labels[0]['org.opencontainers.image.version'],
    sha: labels[0]['org.opencontainers.image.revision'],
  };
}

// docker-plan lets the workflow skip an already published version. docker-promote
// verifies that index again, then points eligible aliases at the same digest.
async function docker(promote) {
  const version = process.env.RELEASE_TAG.slice(1);
  const current = await inspect(version);
  if (current && (current.version !== version || current.sha !== process.env.RELEASE_SHA))
    throw new Error('Conflicting version image');
  output('exists', String(Boolean(current)));
  if (!promote) return;
  if (!current) throw new Error('Version image missing after build');
  // On a fresh build, ensure the version tag still resolves to Buildx's output.
  if (process.env.IMAGE_DIGEST && current.digest !== process.env.IMAGE_DIGEST)
    throw new Error('Built index digest mismatch');
  for (const alias of aliases(version)) {
    const existing = await inspect(alias);
    const action = aliasAction(version, current, existing);
    if (action === 'preserve') continue;
    if (action === 'promote')
      run('docker', [
        'buildx',
        'imagetools',
        'create',
        '--tag',
        `${image}:${alias}`,
        `${image}@${current.digest}`,
      ]);
    if ((await inspect(alias))?.digest !== current.digest)
      throw new Error(`Alias verification failed: ${alias}`);
  }
  output('digest', current.digest);
}

// Importing the pure helpers from tests must not start a publication command.
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const commands = {
    verify,
    restore,
    assets: () => assets(),
    'check-assets': () => assets(true),
    'docker-plan': () => docker(false),
    'docker-promote': () => docker(true),
  };
  try {
    if (!commands[process.argv[2]]) throw new Error('Unknown release command');
    await commands[process.argv[2]]();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
