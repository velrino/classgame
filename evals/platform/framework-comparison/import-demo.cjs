// Import recorded scores into the native Genkit Dev UI without running a model.
const { readFileSync } = require('node:fs');
const { createHash } = require('node:crypto');
const { resolve } = require('node:path');
const { deepStrictEqual, ok } = require('node:assert/strict');
const {
  enrichResultsWithScoring, extractMetricSummaries, getDatasetStore, getEvalStore,
} = require('@genkit-ai/tools-common/eval');

const comparisons = resolve(__dirname, '../../comparisons');
const datasetId = 'mia-water-cycle-20261125';

function readSnapshot(file, hash) {
  const raw = readFileSync(resolve(comparisons, file));
  if (hash) deepStrictEqual(createHash('sha256').update(raw).digest('hex'), hash);
  return JSON.parse(raw);
}

async function main() {
  const report = readSnapshot('2026-11-25-genkit-deepeval/results.json');
  const fixture = readSnapshot('2026-11-25-mia-water-cycle/cases.json', report.dataset_sha256);
  const tutors = readSnapshot('2026-11-25-mia-water-cycle/results.json', report.responses_sha256);
  ok(fixture.synthetic && report.complete, 'Expected a complete synthetic comparison');
  const samples = report.rows.map(({ case: caseId, model }) => {
    const question = fixture.cases.find(c => c.id === caseId);
    const run = tutors.runs.find(r => r.case_id === caseId && r.model_requested === model);
    const context = run.retrieved_chunk_ids.map(id =>
      fixture.mission.knowledge_chunks.find(chunk => chunk.id === id).content);
    return {
      testCaseId: model + '-' + caseId,
      input: { question: question.retrieval.turn.Prompt, tutorModel: model },
      reference: question.expected_sources,
      context,
      output: [run.tutor.message, ...run.tutor.explanation_steps, run.tutor.why_prompt].join('\n'),
      traceIds: [],
    };
  });
  ok(samples.length === 8 && new Set(samples.map(s => s.testCaseId)).size === 8);
  const dataset = samples.map(({ testCaseId, input, reference }) => ({ testCaseId, input, reference }));
  const datasets = getDatasetStore();
  let metadata = (await datasets.listDatasets()).find(d => d.datasetId === datasetId);
  if (metadata) {
    deepStrictEqual(await datasets.getDataset(datasetId), dataset);
  } else {
    metadata = await datasets.createDataset({ datasetId, datasetType: 'UNKNOWN', data: dataset });
  }

  const metricsMetadata = {
    '/evaluator/classgame/source-ids': { displayName: 'Source IDs', definition: fixture.rubric.source_ids },
    '/evaluator/classgame/source-support': { displayName: 'Source support', definition: fixture.rubric.source_only },
  };
  const store = await getEvalStore();
  for (const framework of ['genkit', 'deepeval']) {
    const scores = { '/evaluator/classgame/source-ids': [], '/evaluator/classgame/source-support': [] };
    for (const [i, row] of report.rows.entries()) {
      const testCaseId = samples[i].testCaseId;
      const matched = row.ids[framework];
      const judge = row[framework];
      scores['/evaluator/classgame/source-ids'].push({
        testCaseId, evaluation: { score: matched, status: matched ? 'PASS' : 'FAIL' },
      });
      scores['/evaluator/classgame/source-support'].push({
        testCaseId, evaluation: { score: judge.score, status: judge.status, details: { reasoning: judge.details.reason } },
      });
    }
    const key = {
      evalRunId: 'mia-' + framework + '-20261125',
      actionRef: '/flow/' + framework,
      datasetId, datasetVersion: metadata.version,
      createdAt: report.timestamp.replace(/^(\d{4})(\d{2})(\d{2})T(\d{2})(\d{2})(\d{2})(\d{3})\d*Z$/, '$1-$2-$3T$4:$5:$6.$7Z'),
      metricSummaries: extractMetricSummaries(scores), metricsMetadata,
    };
    const run = { key, results: enrichResultsWithScoring(scores, samples), metricsMetadata };
    await store.save(run);
  }
  console.log('Genkit and DeepEval: 2 recorded runs, 8 answers each. Open http://127.0.0.1:4000/datasets/mia-water-cycle-20261125/evaluate/mia-genkit-20261125--vs--mia-deepeval-20261125');
}
main().catch(error => { console.error(error); process.exitCode = 1; });
