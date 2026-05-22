import { spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

type Options = {
  buildFrontend: boolean;
  outputRoot: string;
  version: string;
};

function parseArgs(args: string[]): Options {
  const options: Options = {
    buildFrontend: false,
    outputRoot: 'dist',
    version: process.env.VERSION || 'dev',
  };

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];

    if (arg === '--build-frontend') {
      options.buildFrontend = true;
      continue;
    }

    if (arg === '--version' || arg === '-v') {
      options.version = readValue(args, index, arg);
      index += 1;
      continue;
    }

    if (arg.startsWith('--version=')) {
      options.version = arg.slice('--version='.length);
      continue;
    }

    if (arg === '--output-root') {
      options.outputRoot = readValue(args, index, arg);
      index += 1;
      continue;
    }

    if (arg.startsWith('--output-root=')) {
      options.outputRoot = arg.slice('--output-root='.length);
      continue;
    }

    if (arg === '--help' || arg === '-h') {
      printUsage();
      process.exit(0);
    }

    throw new Error(`Unknown argument: ${arg}`);
  }

  if (!options.version) {
    throw new Error('Version cannot be empty');
  }

  return options;
}

function readValue(args: string[], index: number, flag: string): string {
  const value = args[index + 1];
  if (!value || value.startsWith('-')) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
}

function printUsage(): void {
  console.log(`Usage: npm run package:windows -- [options]

Options:
  --version, -v <value>      Version embedded into cpa-pc.exe. Defaults to VERSION or dev.
  --output-root <path>       Output root directory. Defaults to dist.
  --build-frontend           Run the web build before packaging.
  --help, -h                 Show this help.
`);
}

function run(
  command: string,
  args: string[],
  options: { cwd: string; env?: NodeJS.ProcessEnv },
): void {
  const result = spawnSync(command, args, {
    cwd: options.cwd,
    env: options.env,
    shell: false,
    stdio: 'inherit',
  });

  if (result.error) {
    throw result.error;
  }

  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status}`);
  }
}

function runNpm(args: string[], options: { cwd: string; env?: NodeJS.ProcessEnv }): void {
  const npmExecPath = process.env.npm_execpath;
  if (npmExecPath && existsSync(npmExecPath)) {
    run(process.execPath, [npmExecPath, ...args], options);
    return;
  }

  throw new Error('npm_execpath is not available; run packaging through npm run package:windows');
}

function commandName(baseName: string): string {
  if (process.platform !== 'win32') {
    return baseName;
  }
  return `${baseName}.exe`;
}

function main(): void {
  const options = parseArgs(process.argv.slice(2));
  const scriptDir = path.dirname(fileURLToPath(import.meta.url));
  const repoRoot = path.dirname(scriptDir);
  const safeVersion = options.version.replace(/[\\/:*?"<>|]/g, '-');
  const packagePath = path.join(repoRoot, options.outputRoot, `cpa-pc_${safeVersion}_windows_amd64`);
  const staticSource = path.join(repoRoot, 'static', 'management.html');
  const configSource = path.join(repoRoot, 'config.example.yaml');

  if (options.buildFrontend) {
    runNpm(['--prefix', path.join(repoRoot, 'web'), 'run', 'build'], {
      cwd: repoRoot,
      env: { ...process.env, VERSION: options.version },
    });
  }

  if (!existsSync(staticSource)) {
    throw new Error('static/management.html not found; run npm --prefix web run build or pass --build-frontend');
  }

  if (!existsSync(configSource)) {
    throw new Error('config.example.yaml not found');
  }

  if (existsSync(packagePath)) {
    rmSync(packagePath, { force: true, recursive: true });
  }

  const staticTargetDir = path.join(packagePath, 'static');
  mkdirSync(staticTargetDir, { recursive: true });
  mkdirSync(path.join(packagePath, 'data'), { recursive: true });
  mkdirSync(path.join(packagePath, 'logs'), { recursive: true });

  run(
    commandName('go'),
    [
      'build',
      '-trimpath',
      '-ldflags',
      `-s -w -X main.version=${options.version}`,
      '-o',
      path.join(packagePath, 'cpa-pc.exe'),
      './cmd/cpa-pc',
    ],
    {
      cwd: repoRoot,
      env: {
        ...process.env,
        CGO_ENABLED: '0',
        GOARCH: 'amd64',
        GOOS: 'windows',
      },
    },
  );

  copyFileSync(configSource, path.join(packagePath, 'config.example.yaml'));
  copyFileSync(staticSource, path.join(staticTargetDir, 'management.html'));

  for (const optionalFile of ['README.md', 'LICENSE']) {
    const source = path.join(repoRoot, optionalFile);
    if (existsSync(source)) {
      copyFileSync(source, path.join(packagePath, optionalFile));
    }
  }

  console.log(`Package created: ${packagePath}`);
}

try {
  main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}
