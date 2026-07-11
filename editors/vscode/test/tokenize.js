// Tokenizes sample Go comment lines with the goboot injection grammar and
// asserts the expected scopes are produced. Run: node test/tokenize.js
const fs = require('fs');
const path = require('path');
const vsctm = require('vscode-textmate');
const oniguruma = require('vscode-oniguruma');

const dir = path.join(__dirname, '..');

function loadJSON(p) {
  return vsctm.parseRawGrammar(fs.readFileSync(p, 'utf8'), p);
}

async function main() {
  const wasm = fs.readFileSync(require.resolve('vscode-oniguruma/release/onig.wasm'));
  await oniguruma.loadWASM(wasm.buffer);
  const onigLib = Promise.resolve({
    createOnigScanner: (patterns) => new oniguruma.OnigScanner(patterns),
    createOnigString: (s) => new oniguruma.OnigString(s),
  });

  const grammars = {
    'source.go': path.join(__dirname, 'go.min.tmLanguage.json'),
    'goboot.annotations.injection': path.join(dir, 'syntaxes', 'goboot.tmLanguage.json'),
  };

  const registry = new vsctm.Registry({
    onigLib,
    loadGrammar: async (scopeName) =>
      grammars[scopeName] ? loadJSON(grammars[scopeName]) : null,
    getInjections: (scopeName) =>
      scopeName === 'source.go' ? ['goboot.annotations.injection'] : [],
  });

  const grammar = await registry.loadGrammar('source.go');

  const samples = [
    '// @Service(name="todoService", implements="TodoUseCase")',
    '// @GetMapping(path="/{id}")',
    '// @Scheduled(fixedRate=2, timeUnit=TimeUnit.MINUTES)',
    '// @Transactional',
    '// @Profile(["prod","staging"])',
    '// @Query(`SELECT id FROM todos WHERE id = :id`)',
    '// the @Transactional method runs in a tx', // prose mention still highlights
  ];

  // token → the goboot scope applied (last matching), for assertions.
  function scopesOf(line) {
    const r = grammar.tokenizeLine(line, vsctm.INITIAL);
    return r.tokens.map((t) => ({
      text: line.substring(t.startIndex, t.endIndex),
      scopes: t.scopes.filter((s) => s.includes('goboot')),
    }));
  }

  let failures = 0;
  function want(line, substr, scopeFragment) {
    const toks = scopesOf(line);
    const hit = toks.find(
      (t) => t.text === substr && t.scopes.some((s) => s.includes(scopeFragment))
    );
    const ok = Boolean(hit);
    if (!ok) failures++;
    console.log(`${ok ? 'PASS' : 'FAIL'}  ${JSON.stringify(substr)} → ${scopeFragment}`);
    if (!ok) {
      console.log('   tokens:', JSON.stringify(toks.filter((t) => t.scopes.length)));
    }
  }

  // Print a full breakdown of the first sample for visibility.
  console.log('--- breakdown:', samples[0]);
  for (const t of scopesOf(samples[0])) {
    if (t.scopes.length) console.log(`   ${JSON.stringify(t.text)}  ${t.scopes.join(', ')}`);
  }
  console.log('--- assertions ---');

  want(samples[0], '@', 'punctuation.definition.annotation');
  want(samples[0], 'Service', 'storage.type.annotation');
  want(samples[0], 'name', 'variable.parameter.annotation');
  want(samples[0], '=', 'keyword.operator.assignment');
  want(samples[0], 'implements', 'variable.parameter.annotation');
  want(samples[1], 'GetMapping', 'storage.type.annotation');
  want(samples[2], 'Scheduled', 'storage.type.annotation');
  want(samples[2], 'TimeUnit.MINUTES', 'constant.other.enum');
  want(samples[2], '2', 'constant.numeric');
  want(samples[3], 'Transactional', 'storage.type.annotation');
  want(samples[4], 'Profile', 'storage.type.annotation');
  want(samples[5], 'Query', 'storage.type.annotation');
  want(samples[6], 'Transactional', 'storage.type.annotation'); // prose mention

  console.log(failures === 0 ? '\nALL PASS' : `\n${failures} FAILURE(S)`);
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
