import React, { useState } from 'react';
import styled from 'styled-components';
import { proxyFetch } from '../utils/proxy_request';

const Container = styled.div`
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
`;

const Header = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e5e7eb;
`;

const Section = styled.div`
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  ${p => p.$overflow && 'overflow: hidden; padding: 0;'}
`;

const Row = styled.div`
  display: flex;
  align-items: center;
  gap: ${p => p.$gap || '16px'};
`;

const Button = styled.button`
  background: ${p => p.$primary ? '#6366f1' : p.$play ? (p.disabled ? '#9ca3af' : '#10b981') : '#f3f4f6'};
  color: ${p => (p.$primary || p.$play) ? '#fff' : '#374151'};
  border: 1px solid ${p => p.$primary ? '#6366f1' : p.$play ? 'transparent' : '#d1d5db'};
  border-radius: ${p => p.$play ? '4px' : '8px'};
  padding: ${p => p.$play ? '6px 12px' : '10px 20px'};
  font-size: ${p => p.$play ? '12px' : '14px'};
  font-weight: 500;
  cursor: ${p => p.disabled ? 'not-allowed' : 'pointer'};
  opacity: ${p => p.disabled ? 0.5 : 1};
  min-width: ${p => p.$play && '60px'};
  height: ${p => p.$play && '28px'};
  transition: all 0.2s ease;
  &:hover:not(:disabled) { filter: brightness(0.95); }
`;

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
  th, td { padding: 12px 16px; text-align: left; font-size: 14px; }
  th { background: #f9fafb; font-weight: 600; color: #374151; }
  tr { border-bottom: 1px solid #e5e7eb; }
  tr:hover { background: #f9fafb; }
`;

const Badge = styled.span`
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  background: ${p => ({ completed: '#dcfce7', error: '#fee2e2', running: '#dbeafe', pending: '#f3f4f6' }[p.$s] || '#f3f4f6')};
  color: ${p => ({ completed: '#166534', error: '#991b1b', running: '#1e40af', pending: '#374151' }[p.$s] || '#374151')};
`;

const ProgressBar = styled.div`
  width: 120px;
  height: 20px;
  background: #e5e7eb;
  border-radius: 10px;
  overflow: hidden;
  div {
    height: 100%;
    background: ${p => ({ completed: '#10b981', error: '#ef4444' }[p.$s] || '#6366f1')};
    width: ${p => p.$w}%;
    transition: width 0.3s ease;
  }
`;

const LogEntry = styled.div`
  padding: 12px;
  margin-bottom: 8px;
  background: #f9fafb;
  border-radius: 8px;
  font-family: Monaco, Consolas, monospace;
  font-size: 13px;
  border-left: 4px solid ${p => ({ error: '#ef4444', success: '#10b981' }[p.$t] || '#6366f1')};
`;

const Link = styled.a`
  color: #6366f1;
  text-decoration: none;
  font-size: 12px;
  &:hover { text-decoration: underline; color: #4f46e5; }
`;

const endpoints = [
  { id: 'coins-list', name: 'Coins List', path: '/v1/coins/list', desc: 'Get all coins list', deps: [] },
  { id: 'coins-markets', name: 'Coins Markets', path: '/v1/coins/markets', desc: 'Get market data (paginated)', deps: [] },
  { id: 'leaderboard-prices', name: 'Leaderboard Prices', path: '/v1/leaderboard/prices', desc: 'Get leaderboard prices', deps: [] },
  { id: 'leaderboard-simple-prices', name: 'Leaderboard Simple Prices', path: '/v1/leaderboard/simpleprices', desc: 'Get simple prices', deps: [] },
  { id: 'leaderboard-markets', name: 'Leaderboard Markets', path: '/v1/leaderboard/markets', desc: 'Get leaderboard markets', deps: [] },
  { id: 'simple-price-markets', name: 'Simple Price (Markets)', path: '/v1/simple/price', desc: 'Prices for market IDs', deps: ['coins-markets'] },
  { id: 'simple-price-coins', name: 'Simple Price (Coins)', path: '/v1/simple/price', desc: 'Prices for coin IDs', deps: ['coins-list'] },
  { id: 'asset-platforms', name: 'Asset Platforms', path: '/v1/asset_platforms', desc: 'Get asset platforms', deps: [] },
  { id: 'coins-markets-by-ids', name: 'Markets for Coins', path: '/v1/coins/markets', desc: 'Market data for coin IDs (chunked)', deps: ['coins-list'] },
  { id: 'token-lists', name: 'Token Lists', path: '/v1/token_lists', desc: 'Token lists for platforms', deps: [] },
  { id: 'coins-id-markets', name: 'Coin Details (Markets)', path: '/v1/coins/{id}', desc: 'Coin details from markets', deps: ['coins-markets'] },
  { id: 'coins-id-list', name: 'Coin Details (List)', path: '/v1/coins/{id}', desc: 'Coin details from list', deps: ['coins-list'] }
];

const TOKEN_PLATFORMS = [
  'ethereum', 'optimistic-ethereum', 'arbitrum-one', 'base', 'linea', 'blast',
  'zksync', 'mantle', 'abstract', 'unichain', 'binance-smart-chain', 'polygon-pos'
];

const EndpointTester = ({ onBack }) => {
  const [marketPages, setMarketPages] = useState(40);
  const [isRunningAll, setIsRunningAll] = useState(false);
  const [status, setStatus] = useState({});
  const [logs, setLogs] = useState([]);
  const [data, setData] = useState({ coinsList: null, marketsData: null });

  const log = (msg, type = 'info') => {
    setLogs(p => [...p, { ts: new Date().toLocaleTimeString(), msg, type }]);
  };

  const upd = (id, s) => {
    setStatus(p => ({ ...p, [id]: { ...p[id], ...s } }));
  };

  const fmt = ms => ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`;
  const delay = ms => new Promise(r => setTimeout(r, ms));

  const buildUrl = (ep) => {
    const base = process.env.REACT_APP_API_URL || 'http://localhost:8080';
    const user = process.env.REACT_APP_PROXY_USER || '';
    const pass = process.env.REACT_APP_PROXY_PASSWORD || '';
    const samples = {
      'coins-markets': '?page=1&per_page=250',
      'simple-price-markets': '?ids=bitcoin,ethereum&vs_currencies=usd',
      'simple-price-coins': '?ids=bitcoin,ethereum&vs_currencies=usd',
      'coins-markets-by-ids': '?ids=bitcoin,ethereum'
    };
    
    let path = ep.path;
    if (ep.id.includes('coins-id')) {
      path = '/v1/coins/bitcoin';
    } else if (ep.id === 'token-lists') {
      path = '/v1/token_lists/ethereum/all.json';
    }
    
    const auth = user && pass ? `${user}:${pass}@` : '';
    return base.replace('://', `://${auth}`) + path + (samples[ep.id] || '');
  };

  const fetchChunked = async (id, ids, pathFn, chunkSize, extract, label) => {
    const start = Date.now();
    upd(id, { status: 'running', progress: 0 });
    
    try {
      const chunks = [];
      for (let i = 0; i < ids.length; i += chunkSize) {
        chunks.push(ids.slice(i, i + chunkSize));
      }
      
      let all = [];
      let total = 0;
      
      for (let i = 0; i < chunks.length; i++) {
        upd(id, { progress: ((i + 1) / chunks.length) * 95 });
        log(`  Chunk ${i + 1}/${chunks.length}: ${chunks[i].length} IDs`, 'info');
        
        const res = await proxyFetch(pathFn(chunks[i]));
        if (!res.ok) {
          throw new Error(`HTTP ${res.status} on chunk ${i + 1}`);
        }
        
        const d = await res.json();
        const count = extract(d);
        all = typeof count === 'number' ? all : [...all, ...d];
        total += typeof count === 'number' ? count : d.length;
        log(`    Received: ${typeof count === 'number' ? count : d.length}/${chunks[i].length}`, 'info');
        
        if (i < chunks.length - 1) {
          await delay(200);
        }
      }
      
      const dur = Date.now() - start;
      const complete = total === ids.length;
      upd(id, { status: 'completed', progress: 100, time: dur, count: total, requested: ids.length, complete });
      log(`${label} - ${total}/${ids.length} ${complete ? '✅' : '⚠️'}`, complete ? 'success' : 'info');
    } catch (e) {
      upd(id, { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
      log(`${label} - Error: ${e.message}`, 'error');
    }
  };

  const exec = {
    'coins-list': async () => {
      const start = Date.now();
      upd('coins-list', { status: 'running', progress: 50 });
      
      try {
        const res = await proxyFetch('/v1/coins/list');
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }
        const d = await res.json();
        setData(p => ({ ...p, coinsList: d }));
        upd('coins-list', { status: 'completed', progress: 100, time: Date.now() - start, count: d.length });
        log(`coins/list - ${d.length} tokens`, 'success');
      } catch (e) {
        upd('coins-list', { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
        log(`coins/list - Error: ${e.message}`, 'error');
      }
    },

    'coins-markets': async () => {
      const start = Date.now();
      upd('coins-markets', { status: 'running', progress: 0 });
      
      try {
        let all = [];
        for (let p = 1; p <= marketPages; p++) {
          upd('coins-markets', { progress: (p / marketPages) * 100 });
          const res = await proxyFetch(`/v1/coins/markets?page=${p}&per_page=250`);
          const d = await res.json();
          
          if (!res.ok || !Array.isArray(d) || !d.length) {
            break;
          }
          all = [...all, ...d];
          log(`  Page ${p}: ${d.length} (total: ${all.length})`, 'info');
        }
        
        setData(p => ({ ...p, marketsData: all }));
        upd('coins-markets', { status: 'completed', progress: 100, time: Date.now() - start, count: all.length });
        log(`coins/markets - ${all.length} tokens`, 'success');
      } catch (e) {
        upd('coins-markets', { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
        log(`coins/markets - Error: ${e.message}`, 'error');
      }
    },

    'simple-price-markets': () => {
      if (!data.marketsData) {
        return log('simple-price-markets - Run coins/markets first', 'error');
      }
      return fetchChunked(
        'simple-price-markets',
        data.marketsData.map(c => c.id),
        ids => `/v1/simple/price?ids=${ids.join(',')}&vs_currencies=usd`,
        500,
        d => Object.keys(d).length,
        'simple/price (markets)'
      );
    },

    'simple-price-coins': () => {
      if (!data.coinsList) {
        return log('simple-price-coins - Run coins/list first', 'error');
      }
      return fetchChunked(
        'simple-price-coins',
        data.coinsList.map(c => c.id),
        ids => `/v1/simple/price?ids=${ids.join(',')}&vs_currencies=usd`,
        500,
        d => Object.keys(d).length,
        'simple/price (coins)'
      );
    },

    'coins-markets-by-ids': () => {
      if (!data.coinsList) {
        return log('coins-markets-by-ids - Run coins/list first', 'error');
      }
      return fetchChunked(
        'coins-markets-by-ids',
        data.coinsList.map(c => c.id),
        ids => `/v1/coins/markets?ids=${ids.join(',')}`,
        250,
        d => d.length,
        'coins/markets (by ids)'
      );
    },

    'token-lists': async () => {
      const start = Date.now();
      upd('token-lists', { status: 'running', progress: 0 });
      
      try {
        let total = 0;
        let ok = 0;
        
        for (let i = 0; i < TOKEN_PLATFORMS.length; i++) {
          upd('token-lists', { progress: ((i + 1) / TOKEN_PLATFORMS.length) * 95 });
          const platform = TOKEN_PLATFORMS[i];
          
          try {
            const res = await proxyFetch(`/v1/token_lists/${platform}/all.json`);
            if (res.ok) {
              const d = await res.json();
              const tokenCount = d.tokens?.length || 0;
              total += tokenCount;
              ok++;
              log(`  ${platform}: ${tokenCount} tokens`, 'success');
            } else {
              log(`  ${platform}: HTTP ${res.status}`, 'error');
            }
          } catch (e) {
            log(`  ${platform}: ${e.message}`, 'error');
          }
          
          if (i < TOKEN_PLATFORMS.length - 1) {
            await delay(100);
          }
        }
        
        const complete = ok === TOKEN_PLATFORMS.length;
        upd('token-lists', {
          status: 'completed',
          progress: 100,
          time: Date.now() - start,
          count: total,
          platforms: ok,
          totalPlatforms: TOKEN_PLATFORMS.length,
          complete
        });
        log(`token_lists - ${ok}/${TOKEN_PLATFORMS.length} platforms, ${total} tokens ${complete ? '✅' : '⚠️'}`, complete ? 'success' : 'info');
      } catch (e) {
        upd('token-lists', { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
        log(`token_lists - Error: ${e.message}`, 'error');
      }
    },

    'coins-id-markets': () => execCoinsId('coins-id-markets', data.marketsData, 'coins/markets'),
    'coins-id-list': () => execCoinsId('coins-id-list', data.coinsList, 'coins/list')
  };

  const execCoinsId = async (id, src, label) => {
    if (!src) {
      return log(`${id} - Run ${label} first`, 'error');
    }
    
    const start = Date.now();
    upd(id, { status: 'running', progress: 0 });
    
    try {
      let ok = 0;
      let stopped = null;
      
      for (let i = 0; i < src.length; i++) {
        upd(id, { progress: ((i + 1) / src.length) * 95 });
        const coinId = src[i].id;
        
        try {
          const res = await proxyFetch(`/v1/coins/${coinId}`);
          
          if (res.ok) {
            ok++;
            if (i < 3) {
              log(`  ${coinId}: OK`, 'success');
            } else if (i === 3) {
              log('  ... continuing', 'info');
            }
          } else {
            stopped = coinId;
            log(`  ${coinId}: HTTP ${res.status} - stopping`, 'error');
            break;
          }
        } catch (e) {
          stopped = coinId;
          log(`  ${coinId}: ${e.message} - stopping`, 'error');
          break;
        }
        
        if (i < src.length - 1) {
          await delay(100);
        }
      }
      
      const complete = !stopped;
      upd(id, {
        status: 'completed',
        progress: 100,
        time: Date.now() - start,
        count: ok,
        requested: src.length,
        errors: stopped ? 1 : 0,
        complete
      });
      
      const stoppedMsg = stopped ? `stopped at '${stopped}'` : '';
      log(`coins/{id} (${label}) - ${ok}/${src.length} ${stoppedMsg} ${complete ? '✅' : '⚠️'}`, complete ? 'success' : 'info');
    } catch (e) {
      upd(id, { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
      log(`coins/{id} (${label}) - Error: ${e.message}`, 'error');
    }
  };

  const execSimple = async (ep) => {
    const start = Date.now();
    upd(ep.id, { status: 'running', progress: 50 });
    
    try {
      const res = await proxyFetch(ep.path);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const d = await res.json();
      const count = Array.isArray(d) ? d.length : Object.keys(d).length;
      upd(ep.id, { status: 'completed', progress: 100, time: Date.now() - start, count });
      log(`${ep.name} - ${count} records`, 'success');
    } catch (e) {
      upd(ep.id, { status: 'error', progress: 100, time: Date.now() - start, error: e.message });
      log(`${ep.name} - Error: ${e.message}`, 'error');
    }
  };

  const run = async (ep) => {
    log(`Starting ${ep.name}...`, 'info');
    if (exec[ep.id]) {
      await exec[ep.id]();
    } else {
      await execSimple(ep);
    }
  };

  const runAll = async () => {
    setIsRunningAll(true);
    setLogs([]);
    setStatus({});
    log('Starting full endpoint test...', 'info');
    
    const order = [
      'coins-list', 'coins-markets', 'leaderboard-prices', 'leaderboard-simple-prices',
      'leaderboard-markets', 'asset-platforms', 'token-lists', 'coins-markets-by-ids',
      'simple-price-markets', 'simple-price-coins', 'coins-id-markets', 'coins-id-list'
    ];
    
    for (const id of order) {
      const ep = endpoints.find(e => e.id === id);
      if (ep) {
        await run(ep);
        await delay(500);
      }
    }
    
    log('All endpoints test completed!', 'success');
    setIsRunningAll(false);
  };

  const getInfo = (ep) => {
    const s = status[ep.id];
    if (!s) {
      return { p: 0, t: 'Pending', s: 'pending' };
    }
    if (s.status === 'running') {
      return { p: s.progress, t: 'Running...', s: 'running' };
    }
    if (s.status === 'error') {
      return { p: 100, t: `Error: ${s.error}`, s: 'error' };
    }
    
    let t = `${fmt(s.time)} - ${s.count} records`;
    if (s.requested !== undefined) {
      t = `${fmt(s.time)} - ${s.count}/${s.requested} ${s.complete ? '✅' : '⚠️'}`;
    }
    if (ep.id === 'token-lists' && s.platforms !== undefined) {
      t = `${fmt(s.time)} - ${s.platforms}/${s.totalPlatforms} platforms, ${s.count} tokens ${s.complete ? '✅' : '⚠️'}`;
    }
    return { p: 100, t, s: 'completed' };
  };

  const canRun = (ep) => {
    return ep.deps.every(d => {
      if (d === 'coins-list') return data.coinsList;
      if (d === 'coins-markets') return data.marketsData;
      return false;
    });
  };

  return (
    <Container>
      <Header>
        <h1 style={{ fontSize: 32, fontWeight: 600, color: '#09101c', margin: 0 }}>Endpoint Tester</h1>
        <Button onClick={onBack}>← Back to Main</Button>
      </Header>

      <Section>
        <Row>
          <label>Market Pages:</label>
          <input
            type="number"
            value={marketPages}
            onChange={e => setMarketPages(+e.target.value || 1)}
            min="1"
            max="100"
            style={{ padding: '8px 12px', border: '1px solid #d1d5db', borderRadius: 6, width: 120 }}
          />
          <Button $primary onClick={runAll} disabled={isRunningAll}>
            {isRunningAll ? 'Running All...' : 'Run All Endpoints'}
          </Button>
        </Row>
      </Section>

      <Section $overflow>
        <Table>
          <thead>
            <tr>
              <th>Play</th>
              <th>Endpoint</th>
              <th>Description</th>
              <th>Progress</th>
              <th>Result</th>
            </tr>
          </thead>
          <tbody>
            {endpoints.map(ep => {
              const info = getInfo(ep);
              const disabled = !canRun(ep) || info.s === 'running' || isRunningAll;
              const title = !canRun(ep) ? `Requires: ${ep.deps.join(', ')}` : 'Run';
              
              return (
                <tr key={ep.id}>
                  <td>
                    <Button $play onClick={() => run(ep)} disabled={disabled} title={title}>
                      ▶
                    </Button>
                  </td>
                  <td>
                    <strong>{ep.name}</strong>
                    <br />
                    <Link href={buildUrl(ep)} target="_blank">{ep.path} ↗</Link>
                  </td>
                  <td>{ep.desc}</td>
                  <td>
                    <Row $gap="8px">
                      <ProgressBar $w={info.p} $s={info.s}><div /></ProgressBar>
                      <Badge $s={info.s}>{info.s}</Badge>
                    </Row>
                  </td>
                  <td>{info.t}</td>
                </tr>
              );
            })}
          </tbody>
        </Table>
      </Section>

      <Section>
        <h3 style={{ fontSize: 18, fontWeight: 600, color: '#374151', margin: '0 0 16px' }}>Execution Log</h3>
        {logs.length === 0 ? (
          <div style={{ color: '#6b7280', fontStyle: 'italic' }}>
            No logs yet. Run an endpoint to see results.
          </div>
        ) : (
          logs.map((l, i) => (
            <LogEntry key={i} $t={l.type}>
              <strong>[{l.ts}]</strong> {l.msg}
            </LogEntry>
          ))
        )}
      </Section>
    </Container>
  );
};

export default EndpointTester;
