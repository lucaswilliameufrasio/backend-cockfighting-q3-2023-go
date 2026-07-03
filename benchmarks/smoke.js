import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const payload = JSON.stringify({
    apelido: `u${__VU}_${__ITER}`,
    nome: 'Test',
    nascimento: '1990-01-01',
    stack: ['go'],
  });

  const create = http.post(`${BASE}/pessoas`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'create-person' },
  });
  check(create, { 'create 201': (r) => r.status === 201 });

  let id = null;
  if (create.status === 201) {
    try { id = create.json('id'); } catch (_) {}
  }

  if (id) {
    const get = http.get(`${BASE}/pessoas/${id}`, { tags: { name: 'get-person' } });
    check(get, { 'get 200': (r) => r.status === 200 });
  }

  const search = http.get(`${BASE}/pessoas?t=u`, { tags: { name: 'search-person' } });
  check(search, { 'search 200': (r) => r.status === 200 });

  if (__ITER % 10 === 0) {
    const count = http.get(`${BASE}/contagem-pessoas`, { tags: { name: 'count-person' } });
    check(count, { 'count 200': (r) => r.status === 200 });
  }

  sleep(0.1);
}
