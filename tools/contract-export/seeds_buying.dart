// Seeds for buying: the demo people, the deals and the friend wishlist.
import 'dart:io';

import 'package:waraqah/features/p2p/data/sources/p2p_people.dart';

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
}
