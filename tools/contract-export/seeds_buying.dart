// Seeds for buying: the demo people, the deals and the friend wishlist.
import 'dart:io';

import 'package:waraqah/features/p2p/data/sources/p2p_fixtures.dart';
import 'package:waraqah/features/p2p/data/sources/p2p_people.dart';
import 'package:waraqah/features/p2p/data/sources/p2p_ratings.dart';

import 'harness.dart';
import 'seeds_catalog.dart' show Get;

Future<void> writeBuyingSeeds(Directory out, Get get) async {
  final dir = out.path;
  // The readers of the demo marketplace; "me" is reader@waraqah.test and is not listed.
  writeJson('$dir/people.json', [
    for (final p in P2pPeople.all)
      if (p.id != P2pPeople.me)
        {
          'id': p.id,
          'name': p.name,
          'area': p.area,
          'district': p.district,
          'memberSince': p.memberSince.toIso8601String(),
          'booksSold': p.booksSold,
        },
  ]);

  // Deals: which Editions are on flash sale at what price, the bundles, the pre-orders.
  final deals = (await get('/deals')) as Map;
  writeJson('$dir/deals.json', {
    'flash': [
      for (final i in (deals['flashSale'] as Map)['items'] as List)
        {'editionId': (i as Map)['editionId'], 'priceBdt': i['priceBdt']},
    ],
    'bundles': [
      for (final b in deals['bundles'] as List)
        {
          'id': (b as Map)['id'],
          'title': b['title'],
          'editionIds': [for (final i in b['items'] as List) (i as Map)['editionId']],
          'priceBdt': b['priceBdt'],
        },
    ],
    'preorders': [
      for (final p in deals['preorders'] as List) ((p as Map)['item'] as Map)['editionId'],
    ],
  });

  // A friend's wishlist, so a shared link can be tried on a fresh start.
  final nabila = (await get('/wishlist/shared', {'id': 'wl-nabila'})) as Map;
  writeJson('$dir/wishlist_shared.json', [
    {
      'id': nabila['id'],
      'ownerId': 'p-nabila',
      'ownerName': nabila['ownerName'],
      'bookIds': [for (final b in nabila['books'] as List) (b as Map)['id']],
    },
  ]);

  // The used marketplace: every seed listing (newest first, "me" is the demo reader), who the
  // reserved and sold ones went to, and the ratings readers gave each other.
  final now = DateTime.now();
  writeJson('$dir/p2p.json', {
    'listings': [
      for (final l in P2pFixtures.listings)
        {
          'id': l.id,
          'title': l.title,
          'sellerId': l.sellerId,
          'priceBdt': l.priceBdt,
          'condition': l.condition.name,
          'flags': l.flags,
          'photos': l.photos,
          'isNegotiable': l.isNegotiable,
          'handover': l.handover.name,
          'status': l.status.name,
          'rejectionReason': l.rejectionReason,
          'bookId': l.bookId,
          'coverSeed': l.coverSeed,
          'district': l.district,
          'area': l.area,
          'categoryId': l.categoryId,
          'newPriceBdt': l.newPriceBdt,
          'note': l.note,
          'buyerId': P2pFixtures.buyers[l.id],
        },
    ],
    'ratings': [
      for (final r in P2pRatingSeed.all(now))
        {
          'fromId': r.fromId,
          'toId': r.toId,
          'stars': r.stars,
          'ageMinutes': now.difference(r.at).inMinutes,
          'listingId': r.listingId,
          'comment': r.comment,
        },
    ],
    'soldBefore': {for (final p in P2pPeople.all) p.id: p.booksSold},
  });
}
