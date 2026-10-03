// Read-only samples (GET) plus the sign-in endpoints. They run first, before any
// sample changes the fake backend's state.
import 'harness.dart';

const _tiny = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==';

List<S> readSamples() => [
  // auth ---------------------------------------------------------------
  S.post(
    '/auth/login',
    body: (_) => {'email': 'reader@waraqah.test', 'password': 'Waraqah#Demo1'},
    as: 'guest',
  ),
  S.post('/auth/google', body: (_) => {'idToken': 'sample-id-token'}, as: 'guest'),
  S.post(
    '/auth/signup/request-otp',
    body: (_) => {
      'name': 'New Reader',
      'contact': 'newreader@waraqah.test',
      'password': 'Secret#123',
    },
    as: 'guest',
  ),
  S.post(
    '/auth/signup/verify-otp',
    body: (_) => {'contact': 'newreader@waraqah.test', 'otp': '123456'},
    as: 'guest',
  ),
  S.post(
    '/auth/signup/verify-otp',
    body: (_) => {'contact': 'newreader@waraqah.test', 'otp': '000000'},
    as: 'guest',
    variant: 'miss',
  ),
  S.post(
    '/auth/password/request-otp',
    body: (_) => {'contact': 'reader@waraqah.test'},
    as: 'guest',
  ),
  S.post(
    '/auth/password/reset',
    body: (_) => {
      'contact': 'reader@waraqah.test',
      'otp': '123456',
      'password': 'Waraqah#Demo1',
    },
    as: 'guest',
  ),
  // catalog ------------------------------------------------------------
  S.get('/books', query: (_) => {'q': 'clean'}),
  S.get('/books', query: (_) => {'includeHidden': 'true'}, as: 'catalog', variant: 'hidden'),
  S.get('/books/detail', query: (_) => {'id': 'bk-cleancode'}),
  S.get('/books/detail', query: (_) => {'id': 'bk-nope'}, variant: 'miss'),
  S.get('/books/details', query: (_) => {'id': 'bk-sapiens'}),
  S.get('/books/look-inside', query: (_) => {'id': 'bk-sherlock'}),
  S.get('/books/price-lows', query: (_) => {'id': 'bk-sapiens'}),
  S.get('/books/series', query: (_) => {'id': 'bk-hpstone'}),
  S.get('/books/used-options', query: (_) => {'id': 'bk-cleancode'}),
  S.get('/books/suggest', query: (_) => {'q': 'sap'}),
  S.get('/books/did-you-mean', query: (_) => {'q': 'sapeins'}),
  S.get('/books/questions', query: (_) => {'id': 'bk-cleancode'}),
  S.get('/categories', query: (_) => {'section': 'academic'}),
  S.get('/subjects'),
  S.get('/authors/detail', query: (_) => {'id': 'au-harari'}),
  S.get('/publishers/detail', query: (_) => {'id': 'pub-harper'}),
  S.get('/series/detail', query: (_) => {'id': 'ser-harry-potter'}),
  S.get('/collections'),
  S.get(
    '/collections/detail',
    query: (c) => {'id': c.id('GET /collections')},
  ),
  S.get('/experts'),
  S.get('/experts/detail', query: (c) => {'id': c.id('GET /experts')}),
  S.get('/booklists', as: 'reader'),
  S.get(
    '/booklists/detail',
    query: (c) => {'id': c.id('GET /booklists')},
  ),
  // home, scan, deals --------------------------------------------------
  S.get('/home/banners'),
  S.get('/home/season'),
  S.get('/islamic/ayah-of-the-day'),
  S.get('/scan/lookup', query: (_) => {'isbn': '9789840001491'}),
  S.get('/deals'),
  // profile, notifications ----------------------------------------------
  S.get('/profile', as: 'reader'),
  S.get('/profile/prefs', as: 'reader'),
  S.get('/addresses', as: 'reader'),
  S.get('/geo'),
  S.get('/notifications', as: 'reader'),
  // guest-friendly me endpoints ------------------------------------------
  S.get('/cart', as: 'reader'),
  S.get('/wishlist', as: 'reader'),
  S.get('/wishlist/shared', query: (_) => {'id': 'wl-nabila'}),
  S.get('/alerts', as: 'reader'),
  S.get('/orders', as: 'reader'),
  S.get('/orders/details', query: (_) => {'number': 'WQ-100215'}, as: 'reader'),
  S.get('/wallet', as: 'reader'),
  S.get('/points', as: 'reader'),
  S.get('/coupons/check', query: (_) => {'code': 'EID100'}, as: 'reader'),
  S.get('/coupons/check', query: (_) => {'code': 'NOPE'}, as: 'reader', variant: 'miss'),
  // donate ----------------------------------------------------------------
  S.get('/donate/recipients'),
  S.get('/donate/recipient', query: (_) => {'id': 'rc-aloghar'}),
  // marketplace --------------------------------------------------------------
  S.get('/p2p/listings'),
  S.get('/p2p/listing', query: (_) => {'id': 'p2p-1'}),
  S.get('/p2p/listing', query: (_) => {'id': 'p2p-nope'}, variant: 'miss'),
  S.get('/p2p/listings/for-book', query: (_) => {'bookId': 'bk-cleancode'}),
  S.get('/p2p/listings/mine', as: 'reader'),
  S.get('/p2p/seller', query: (_) => {'id': 'p-nabila'}),
  S.get('/blocks', as: 'reader'),
  S.get('/inbox', as: 'reader'),
  S.get(
    '/inbox/thread',
    query: (c) => {'id': c.id('GET /inbox')},
    as: 'reader',
  ),
  S.get('/inbox/live', as: 'reader', sse: true),
  S.get('/requests/mine', as: 'reader'),
  S.get('/requests/wanted', as: 'reader'),
  S.get('/requests/demand', as: 'support'),
  S.get('/sales/mine', as: 'reader'),
  S.get('/sales/detail', query: (_) => {'id': 'HS-101'}, as: 'reader'),
  S.get('/sales/earnings', as: 'reader'),
  S.get('/sales/live', as: 'reader', sse: true),
  S.get('/notifications/live', as: 'reader', sse: true),
  S.get('/sell-back/books', query: (_) => {'q': 'zero'}, as: 'reader'),
  S.get('/sell-back/book', query: (_) => {'id': 'bk-zero'}, as: 'reader'),
  S.get('/sell-back/mine', as: 'reader'),
  // community ----------------------------------------------------------------
  S.get('/bites', query: (_) => {'feed': 'forYou'}),
  S.get('/bites/detail', query: (c) => {'id': c.id('GET /bites')}),
  S.get('/reviews', query: (_) => {'bookId': 'bk-cleancode'}),
  S.get('/readers/detail', query: (_) => {'id': 'p-nabila'}),
  S.get('/shelves', as: 'reader'),
  S.get('/reading/stats', as: 'reader'),
  // assistant + staff reads ---------------------------------------------------
  S.get('/assistant/greeting', query: (_) => {'lang': 'en'}),
  S.get('/admin/dashboard', as: 'admin'),
  S.get('/admin/orders', as: 'support'),
  S.get('/admin/coupons', as: 'support'),
  S.get('/admin/catalog/banners', as: 'catalog'),
  S.get('/admin/catalog/season', as: 'catalog'),
  S.get('/admin/catalog/categories', as: 'catalog'),
  S.get('/admin/catalog/authors', as: 'catalog'),
  S.get('/admin/catalog/publishers', as: 'catalog'),
  S.get('/admin/catalog/low-stock', as: 'catalog'),
  S.get('/admin/catalog/isbn-lookup', query: (_) => {'isbn': '9789840001491'}, as: 'catalog'),
  S.get('/moderation/listings', as: 'moderator'),
  S.get('/moderation/reports', as: 'moderator'),
  S.get('/moderation/log', as: 'moderator'),
  S.get('/sales/disputes', as: 'moderator'),
  S.get('/sell-back/queue', as: 'catalog'),
];

/// A one-pixel PNG, base64, for the sample photo uploads.
const samplePhoto = _tiny;
