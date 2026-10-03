// Catalog seeds: books, the records they point at, collections, booklists and the extras.
import 'dart:io';

import 'package:waraqah/features/catalog/data/sources/author_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/book_fake_api.dart';
import 'package:waraqah/features/catalog/data/sources/book_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/book_question_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/book_sort.dart';
import 'package:waraqah/features/catalog/data/sources/booklist_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/category_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/collection_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/expert_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/look_inside_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/publisher_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/seed/about_seed.dart';
import 'package:waraqah/features/catalog/data/sources/series_fixtures.dart';
import 'package:waraqah/features/catalog/data/sources/subject_fixtures.dart';
import 'package:waraqah/features/catalog_admin/data/sources/isbn_lookup_fixtures.dart';
import 'package:waraqah/features/home/data/sources/ayah_fixtures.dart';
import 'package:waraqah/features/home/data/sources/banner_fixtures.dart';

import 'harness.dart';

typedef Get = Future<dynamic> Function(String path, [Map<String, dynamic>? query]);

Future<void> writeCatalogSeeds(Directory out, Get get, DateTime now) async {
  final dir = out.path;
  // Every Book, hidden ones too, in storefront order (the order of /books with no sort).
  writeJson('$dir/books.json', plain([for (final b in BookFixtures.all) b.toJson()]));
  writeJson('$dir/categories.json', plain([for (final c in CategoryFixtures.all) c.toJson()]));
  writeJson('$dir/authors.json', plain([for (final a in AuthorFixtures.all) a.toJson()]));
  writeJson('$dir/publishers.json', plain([for (final p in PublisherFixtures.all) p.toJson()]));
  writeJson('$dir/subjects.json', plain([for (final s in SubjectFixtures.all) s.toJson()]));
  writeJson('$dir/collections.json', plain([for (final c in CollectionFixtures.all) c.toJson()]));
  writeJson('$dir/experts.json', plain([for (final e in ExpertFixtures.all) e.toJson()]));
  writeJson('$dir/booklists.json', plain([for (final b in BooklistFixtures.all) b.toJson()]));
  writeJson('$dir/series.json', plain([for (final s in SeriesFixtures.all) s.toJson()]));
  writeJson('$dir/book_details.json', [
    for (final e in AboutSeed.byBookId.entries)
      {'bookId': e.key, 'description': e.value.$1, 'pages': e.value.$2},
  ]);
  writeJson('$dir/look_inside.json', [
    for (final e in LookInsideFixtures.byBook.entries) {'bookId': e.key, ...plain(e.value.toJson()) as Map},
  ]);
  // Questions are dated relative to now: kept as ages in minutes.
  int age(DateTime t) => now.difference(t).inMinutes;
  writeJson('$dir/book_questions.json', [
    for (final e in BookQuestionFixtures.seed(now).entries)
      for (final q in e.value)
        {
          'bookId': e.key,
          'id': q.id,
          'text': q.text,
          'askerName': q.askerName,
          'ageMinutes': age(q.askedAt),
          'answers': [
            for (final a in q.answers)
              {
                'id': a.id,
                'text': a.text,
                'authorName': a.authorName,
                'ageMinutes': age(a.answeredAt),
                'isStaff': a.isStaff,
              },
          ],
        },
  ]);
  // Copies sold in the last 30 days per Edition (bestselling sort) and the cheaper earlier
  // prices the fake API remembers (the 30-day low badge).
  writeJson('$dir/sales_30d.json', BookSort.sales30Days);
  final lows = <Map<String, Object?>>[];
  for (final b in BookFixtures.all) {
    final answer = (await get(BookFakeApi.priceLows, {'id': b.id})) as Map;
    for (final e in b.editions) {
      final low = answer[e.id] as int;
      if (low < e.priceBdt) lows.add({'editionId': e.id, 'lowBdt': low});
    }
  }
  writeJson('$dir/price_lows.json', lows);
  // Home: Banners in display order, the verses Home rotates through, and the outside books the
  // ISBN lookup knows.
  writeJson('$dir/banners.json', plain([for (final b in BannerFixtures.all) b.toJson()]));
  writeJson('$dir/ayahs.json', plain([for (final a in AyahFixtures.verses) a.toJson()]));
  writeJson('$dir/isbn_lookup.json', [
    for (final e in IsbnLookupFixtures.byIsbn.entries) {'isbn': e.key, ...e.value},
  ]);
}
