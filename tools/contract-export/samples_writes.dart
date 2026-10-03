// Samples that change the fake backend. They run after the reads, in an order that makes
// each one meaningful (a cart before checkout, a listing before moderation, ...).
import 'harness.dart';
import 'samples_reads.dart' show samplePhoto;

String _editionOf(Ctx c, String bookId) {
  final book = c.of('GET /books/detail') as Map;
  return '${((book['editions'] as List).first as Map)['id']}';
}

List<S> writeSamples() => [
  // profile ---------------------------------------------------------------
  S.post('/profile/save', body: (_) => {'name': 'Rafiq Ahmed', 'phone': '01712345678'}),
  S.post('/profile/prefs/save', body: (_) => {'muted': <String>[], 'profileVisible': true, 'activityVisible': true}),
  S.post(
    '/addresses/save',
    body: (_) => {
      'label': 'Office',
      'recipient': 'Rafiq',
      'phone': '01712345678',
      'line': 'Level 4, Gulshan Avenue',
      'upazila': 'Dhaka North City',
      'district': 'Dhaka',
      'division': 'Dhaka',
    },
  ),
  S.post('/addresses/default', body: (c) => {'id': 'addr-family'}),
  S.post('/addresses/delete', body: (c) => {'id': 'addr-family'}),
  S.post('/addresses/delete', body: (c) => {'id': 'addr-nope'}, variant: 'miss'),
  S.post('/notifications/read', body: (c) => {'id': c.id('GET /notifications')}),
  S.post('/notifications/read-all'),
  // catalog (reader) ----------------------------------------------------------
  S.post(
    '/books/questions/ask',
    body: (_) => {'bookId': 'bk-cleancode', 'text': 'Is this the second edition?', 'name': 'Rafiq'},
  ),
  S.post(
    '/books/questions/answer',
    body: (c) => {
      'bookId': 'bk-cleancode',
      'questionId': c.of('GET /books/questions') is List && (c.of('GET /books/questions') as List).isNotEmpty
          ? c.id('GET /books/questions')
          : 'q-1',
      'text': 'Yes, it is.',
      'name': 'Rafiq',
      'isStaff': false,
    },
  ),
  S.post('/booklists/mine/save', body: (_) => {'name': 'My weekend reads', 'bookIds': ['bk-cleancode', 'bk-sapiens']}),
  S.post('/booklists/mine/delete', body: (c) => {'id': c.id('POST /booklists/mine/save')}),
  S.post('/booklists/mine/delete', body: (_) => {'id': 'bl-staff-1'}, variant: 'miss'),
  // wishlist, alerts, cart --------------------------------------------------------
  S.post('/wishlist/save', body: (_) => {'bookId': 'bk-sapiens'}),
  S.post('/wishlist/share', body: (_) => {'ownerName': 'Rafiq'}),
  S.post('/wishlist/remove', body: (_) => {'bookId': 'bk-sapiens'}),
  S.post(
    '/alerts/set',
    body: (_) => {'kind': 'priceDrop', 'bookId': 'bk-atomic', 'editionId': 'bk-atomic-pb-en', 'targetPriceBdt': 500},
  ),
  S.post('/alerts/remove', body: (c) => {'id': c.id('POST /alerts/set')}),
  S.post('/cart/add', body: (_) => {'kind': 'edition', 'id': 'bk-atomic-pb-en'}),
  S.post('/cart/add', body: (_) => {'kind': 'edition', 'id': 'bk-sapiens-pb-en'}),
  S.post(
    '/cart/update',
    body: (c) => {'lineId': c.id('POST /cart/add', inList: 'lines'), 'quantity': 2},
  ),
  S.post(
    '/cart/remove',
    body: (c) => {'lineId': c.id('POST /cart/add', inList: 'lines', last: true)},
  ),
  // checkout and orders ------------------------------------------------------------
  S.post(
    '/orders/place',
    body: (_) => {
      'addressId': 'addr-home',
      'payment': 'cashOnDelivery',
      'couponCode': 'EID100',
      'usePoints': false,
      'useWallet': false,
    },
  ),
  S.post(
    '/orders/place',
    body: (_) => {'addressId': 'addr-home', 'payment': 'cashOnDelivery', 'usePoints': false, 'useWallet': false},
    variant: 'miss',
  ),
  S.post('/orders/cancel', body: (c) => {'number': '${(c.of('POST /orders/place') as Map)['number']}'}),
  S.post('/orders/reorder', body: (_) => {'number': 'WQ-100201'}),
  S.post(
    '/orders/return',
    body: (_) => {'number': 'WQ-100201', 'reason': 'damaged', 'note': 'Torn cover', 'photos': [samplePhoto]},
  ),
  S.post('/admin/orders/advance', body: (_) => {'number': 'WQ-100215', 'status': 'delivered'}, as: 'support'),
  S.post('/admin/orders/return', body: (_) => {'number': 'WQ-100201', 'approve': true}, as: 'support'),
  S.post(
    '/admin/coupons/create',
    body: (_) => {'code': 'SAMPLE50', 'kind': 'amountOff', 'value': 50, 'minOrderBdt': 300},
    as: 'support',
  ),
  S.post(
    '/donate/give',
    body: (_) => {
      'recipientId': 'rc-aloghar',
      'bookId': 'bk-hobbit',
      'quantity': 1,
      'payment': 'bkash',
      'note': 'Happy reading',
    },
  ),
  S.post(
    '/admin/donate/places/save',
    body: (_) => {
      'name': 'Sample Reading Corner',
      'kind': 'library',
      'district': 'Dhaka',
      'area': 'Mirpur',
      'story': 'A small reading corner run by volunteers.',
      'needs': [
        {'bookId': 'bk-sapiens', 'wanted': 5},
      ],
    },
    as: 'support',
  ),
  S.post(
    '/admin/donate/places/remove',
    body: (c) => {'id': c.id('POST /admin/donate/places/save', last: true)},
    as: 'support',
  ),
  // marketplace --------------------------------------------------------------------------
  S.post(
    '/p2p/listings/save',
    body: (_) => {
      'submit': true,
      'title': 'Clean Code',
      'bookId': 'bk-cleancode',
      'condition': 'good',
      'flags': <String>[],
      'priceBdt': 600,
      'isNegotiable': true,
      'handover': 'meetInPerson',
      'photos': ['front', 'back'],
      'photoData': {'front': samplePhoto, 'back': samplePhoto},
    },
  ),
  S.post('/moderation/listings/decide', body: (c) => {'listingId': c.id('POST /p2p/listings/save'), 'decision': 'approve', 'by': 'ignored'}, as: 'moderator'),
  S.post('/moderation/listings/decide', body: (_) => {'listingId': 'p2p-review-1', 'decision': 'reject', 'reason': 'Photos are blurry', 'by': 'ignored'}, as: 'moderator', variant: 'reject'),
  S.post('/reports', body: (_) => {'kind': 'listing', 'targetId': 'p2p-1', 'reason': 'spam', 'note': 'Looks fake'}),
  S.post('/moderation/reports/act', body: (c) => {'reportId': c.id('POST /reports'), 'action': 'dismiss', 'by': 'ignored'}, as: 'moderator'),
  S.post('/blocks/add', body: (_) => {'readerId': 'p-rakib'}),
  S.post('/blocks/remove', body: (_) => {'readerId': 'p-rakib'}),
  // inbox ----------------------------------------------------------------------------------------
  S.post('/inbox/open', body: (_) => {'listingId': 'p2p-1'}),
  S.post('/inbox/send', body: (c) => {'threadId': c.id('POST /inbox/open'), 'text': 'Is it still available?'}),
  S.post('/inbox/offer', body: (c) => {'listingId': 'p2p-1', 'amountBdt': (c.of('GET /p2p/listing') as Map)['priceBdt'], 'handover': 'meetup'}),
  S.post('/inbox/offer/decide', body: (_) => {'threadId': 'th-sadia', 'offerId': 'o-sadia', 'accept': true}),
  S.post('/inbox/read', body: (c) => {'threadId': c.id('POST /inbox/open')}),
  S.post('/inbox/listing/sold', body: (_) => {'threadId': 'th-sadia'}),
  S.post('/inbox/rate', body: (_) => {'threadId': 'th-sadia', 'stars': 5, 'comment': 'Smooth deal'}),
  // After the sale the book can no longer be released: the refusal is the sample.
  S.post('/inbox/listing/release', body: (_) => {'threadId': 'th-sadia'}, refusal: true),
  // requests -----------------------------------------------------------------------------------------
  S.post('/requests', body: (_) => {'title': 'The Pragmatic Programmer', 'author': 'Andrew Hunt', 'maxPriceBdt': 700, 'note': 'Any edition'}),
  S.post('/requests/close', body: (c) => {'id': c.id('POST /requests')}),
  // handled sales -----------------------------------------------------------------------------------------
  S.post('/sales/buy', body: (_) => {'listingId': 'p2p-1', 'method': 'bkash'}),
  S.post('/sales/step', body: (_) => {'id': 'HS-103', 'step': 'send'}),
  S.post('/sales/step', body: (c) => {'id': c.id('POST /sales/buy'), 'step': 'cancel'}, variant: 'cancel'),
  S.post('/sales/dispute', body: (_) => {'id': 'HS-101', 'reason': 'notAsDescribed', 'note': 'Water damage', 'photos': [samplePhoto]}),
  S.post('/sales/disputes/settle', body: (_) => {'id': 'HS-101', 'refund': true, 'by': 'ignored'}, as: 'moderator'),
  S.post('/sales/payout', refusal: true),
  // sell back ------------------------------------------------------------------------------------------------
  S.post('/sell-back', body: (_) => {'bookId': 'bk-zero', 'condition': 'good', 'flags': 0, 'pickupAddress': 'House 12, Road 5, Dhanmondi, Dhaka'}),
  S.post('/sell-back/grade', body: (c) => {'id': c.id('POST /sell-back'), 'condition': 'good', 'accept': true, 'by': 'ignored'}, as: 'catalog'),
  // community --------------------------------------------------------------------------------------------------
  S.post('/bites/post', body: (_) => {'text': 'Finished Clean Code today, worth it.', 'bookId': 'bk-cleancode', 'spoiler': false}),
  S.post('/bites/edit', body: (c) => {'id': c.id('POST /bites/post'), 'text': 'Finished Clean Code today.', 'spoiler': false}),
  S.post('/bites/like', body: (c) => {'id': c.id('POST /bites/post'), 'liked': true}),
  S.post('/bites/comments/post', body: (c) => {'biteId': c.id('POST /bites/post'), 'text': 'Nice one'}),
  S.post('/bites/comments/delete', body: (c) => {'id': c.id('POST /bites/comments/post', inList: 'comments', last: true)}),
  S.post('/bites/delete', body: (c) => {'id': c.id('POST /bites/post')}),
  S.post('/reviews/save', body: (_) => {'bookId': 'bk-cleancode', 'stars': 5, 'text': 'A must-read for programmers.'}),
  S.post('/reviews/delete', body: (_) => {'bookId': 'bk-cleancode'}),
  S.post('/readers/follow', body: (_) => {'id': 'p-nabila', 'follow': true}),
  S.post('/shelves/move', body: (_) => {'bookId': 'bk-cleancode', 'shelf': 'reading'}),
  S.post('/shelves/progress', body: (_) => {'bookId': 'bk-cleancode', 'percent': 40, 'pagesRead': 180, 'totalPages': 450}),
  S.post('/reading/goal', body: (_) => {'goal': 24}),
  // assistant ----------------------------------------------------------------------------------------------------
  S.post('/assistant/ask', body: (_) => {'prompt': 'Suggest a book about habits', 'history': <String>[], 'lang': 'en'}),
  // catalog admin ----------------------------------------------------------------------------------------------------
  S.post(
    '/admin/catalog/books/save',
    body: (_) => {
      'title': 'Sample Staff Book',
      'titleBn': '',
      'authorId': 'au-harari',
      'publisherId': 'pub-harper',
      'section': 'academic',
      'categoryId': 'cat-academic',
      'originalLanguage': 'english',
      'coverSeed': 3,
      'editions': [
        {'id': '', 'format': 'paperback', 'language': 'english', 'priceBdt': 400, 'stock': 10},
      ],
      'classes': <int>[],
      'exams': <String>[],
      'subjectId': '',
    },
    as: 'catalog',
  ),
  S.post('/admin/catalog/books/hide', body: (c) => {'id': c.id('POST /admin/catalog/books/save'), 'hidden': true}, as: 'catalog'),
  S.post('/admin/catalog/categories/save', body: (_) => {'name': 'Sample Category', 'nameBn': 'নমুনা', 'section': 'academic'}, as: 'catalog'),
  S.post('/admin/catalog/categories/delete', body: (c) => {'id': c.id('POST /admin/catalog/categories/save', last: true)}, as: 'catalog'),
  S.post('/admin/catalog/authors/save', body: (_) => {'name': 'Sample Author', 'nameBn': ''}, as: 'catalog'),
  S.post('/admin/catalog/authors/delete', body: (c) => {'id': c.id('POST /admin/catalog/authors/save', last: true)}, as: 'catalog'),
  S.post('/admin/catalog/publishers/save', body: (_) => {'name': 'Sample Publisher', 'nameBn': ''}, as: 'catalog'),
  S.post('/admin/catalog/publishers/delete', body: (c) => {'id': c.id('POST /admin/catalog/publishers/save', last: true)}, as: 'catalog'),
  S.post(
    '/admin/catalog/banners/save',
    body: (_) => {
      'id': '',
      'titleEn': 'Sample banner',
      'titleBn': 'নমুনা ব্যানার',
      'subtitleEn': 'Sample subtitle',
      'subtitleBn': '',
      'seed': 2,
      'target': {'kind': 'book', 'value': 'bk-cleancode'},
      'season': null,
    },
    as: 'catalog',
  ),
  S.post('/admin/catalog/banners/move', body: (c) => {'id': c.id('POST /admin/catalog/banners/save', last: true), 'by': -1}, as: 'catalog'),
  S.post('/admin/catalog/banners/delete', body: (c) => {'id': c.id('POST /admin/catalog/banners/save', last: true)}, as: 'catalog'),
  S.post('/admin/catalog/season/save', body: (_) => {'season': 'ramadan'}, as: 'catalog'),
  S.post(
    '/admin/catalog/collections/save',
    body: (_) => {'titleEn': 'Sample collection', 'titleBn': 'নমুনা সংগ্রহ', 'noteEn': 'Sample', 'noteBn': '', 'section': 'academic', 'bookIds': ['bk-cleancode']},
    as: 'catalog',
  ),
  S.post('/admin/catalog/collections/delete', body: (c) => {'id': c.id('POST /admin/catalog/collections/save')}, as: 'catalog'),
  S.post(
    '/admin/catalog/booklists/save',
    body: (_) => {'titleEn': 'Sample staff list', 'titleBn': 'নমুনা তালিকা', 'noteEn': 'Sample', 'noteBn': '', 'kind': 'classList', 'bookIds': ['bk-cleancode']},
    as: 'catalog',
  ),
  S.post('/admin/catalog/booklists/delete', body: (c) => {'id': c.id('POST /admin/catalog/booklists/save')}, as: 'catalog'),
  S.post('/admin/catalog/editions/stock', body: (_) => {'editionId': 'bk-atomic-pb-en', 'stock': 3}, as: 'catalog'),
  S.post(
    '/admin/catalog/import',
    body: (_) => {
      'books': [
        {
          'row': 2,
          'author': 'Sample Author',
          'publisher': 'Sample Publisher',
          'title': 'Imported Sample',
          'titleBn': '',
          'authorId': '',
          'publisherId': '',
          'section': 'academic',
          'categoryId': 'cat-academic',
          'originalLanguage': 'english',
          'coverSeed': 1,
          'editions': [
            {'id': '', 'format': 'paperback', 'language': 'english', 'priceBdt': 300, 'stock': 5},
          ],
          'classes': <int>[],
          'exams': <String>[],
          'subjectId': '',
        },
      ],
    },
    as: 'catalog',
  ),
  // The account ends here, so this is the last sample.
  S.post('/auth/delete', body: (_) => <String, dynamic>{}),
];
