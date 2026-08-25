// Currencies the proxy can convert to, mirroring currency_ratios.currencies in
// market-fetcher/config.yaml. Anything outside this list is answered with 400.
export const SUPPORTED_CURRENCIES = [
  'usd', 'aed', 'ars', 'aud', 'bdt', 'bhd', 'bmd', 'brl', 'cad', 'chf',
  'clp', 'cny', 'czk', 'dkk', 'eur', 'gbp', 'gel', 'hkd', 'huf', 'idr',
  'ils', 'inr', 'jpy', 'krw', 'kwd', 'lkr', 'mmk', 'mxn', 'myr', 'ngn',
  'nok', 'nzd', 'php', 'pkr', 'pln', 'rub', 'sar', 'sek', 'sgd', 'thb',
  'try', 'twd', 'uah', 'vnd', 'zar', 'xdr', 'btc', 'eth'
];

// BASE_CURRENCY is what the proxy caches as Passthrough; requesting it needs no
// conversion parameter.
export const BASE_CURRENCY = 'usd';

// Currencies Intl cannot format as an ISO-4217 amount: crypto and the IMF unit.
const NON_ISO_CURRENCIES = new Set(['btc', 'eth', 'xdr']);

// convertParam returns the query fragment requesting a converted response, empty
// for the base currency.
export function convertParam(currency) {
  if (!currency || currency === BASE_CURRENCY) {
    return '';
  }
  return `convert_currency=${currency}`;
}

// formatAmount renders a number in the given currency, falling back to a plain
// number plus a code suffix where Intl has no currency formatting.
export function formatAmount(value, currency = BASE_CURRENCY) {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '—';
  }

  if (NON_ISO_CURRENCIES.has(currency)) {
    const digits = currency === 'xdr' ? 2 : 8;
    const amount = new Intl.NumberFormat('en-US', {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits
    }).format(value);
    return `${amount} ${currency.toUpperCase()}`;
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value);
}
