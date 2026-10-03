// Seed files: what the backend loads into Postgres so the app shows the same data it shows
// on the fake API. Each feature task adds its writers here (BACKEND_PLAN.md section 14).
//
// Writers read a fresh fake backend (new stores, nothing changed yet), so the seeds are the
// starting state the fake API shows a new reader.
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:waraqah/app/fake_api_routes.dart';
import 'package:waraqah/app/fake_stores.dart';
import 'package:waraqah/features/profile/data/sources/geo/bd_geo.dart';

import 'harness.dart';

Future<void> writeSeeds(Directory out) async {
  out.createSync(recursive: true);
  final dio = Dio()..interceptors.add(FakeApiRoutes.interceptor(FakeStores()));
  final now = DateTime.now();

  Future<dynamic> get(String path, [Map<String, dynamic>? query]) async =>
      plain((await dio.get<dynamic>(path, queryParameters: query)).data);

  // The demo accounts. The fake API's signed-in user "me" is reader@waraqah.test.
  writeJson('${out.path}/users.json', [
    {'id': 'u_admin', 'email': 'admin@waraqah.test', 'name': 'Waraqah Admin', 'role': 'superAdmin'},
    {'id': 'u_moderator', 'email': 'moderator@waraqah.test', 'name': 'Waraqah Moderator', 'role': 'moderator'},
    {'id': 'u_catalog', 'email': 'catalog@waraqah.test', 'name': 'Catalog Manager', 'role': 'catalogManager'},
    {'id': 'u_support', 'email': 'support@waraqah.test', 'name': 'Waraqah Support', 'role': 'support'},
    {'id': 'u_reader', 'email': 'reader@waraqah.test', 'name': 'Reader', 'role': 'reader'},
  ]);

  // Profile: the divisions, districts and upazilas, and the reader's saved addresses.
  writeJson('${out.path}/geo.json', plain(BdGeo.toJson()));
  writeJson('${out.path}/addresses.json', await get('/addresses'));

  // Notifications: times are relative to the moment of export, so they are kept as an age and
  // the loader counts back from the moment of seeding.
  final notes = (await get('/notifications')) as List;
  writeJson('${out.path}/notifications.json', [
    for (final n in notes)
      {
        for (final e in (n as Map).entries)
          if (e.key != 'createdAt') e.key as String: e.value,
        'ageMinutes': now.difference(DateTime.parse(n['createdAt'] as String)).inMinutes,
      },
  ]);
}
