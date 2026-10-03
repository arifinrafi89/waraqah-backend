// Seed files: what the backend loads into Postgres so the app shows the same data it shows
// on the fake API. Each feature task adds its writers here (BACKEND_PLAN.md section 14).
import 'dart:io';

import 'package:dio/dio.dart';

import 'harness.dart';

Future<void> writeSeeds(Dio dio, Directory out) async {
  out.createSync(recursive: true);
  // The demo accounts. The fake API's signed-in user "me" is reader@waraqah.test.
  writeJson('${out.path}/users.json', [
    {'id': 'u_admin', 'email': 'admin@waraqah.test', 'name': 'Waraqah Admin', 'role': 'superAdmin'},
    {'id': 'u_moderator', 'email': 'moderator@waraqah.test', 'name': 'Waraqah Moderator', 'role': 'moderator'},
    {'id': 'u_catalog', 'email': 'catalog@waraqah.test', 'name': 'Catalog Manager', 'role': 'catalogManager'},
    {'id': 'u_support', 'email': 'support@waraqah.test', 'name': 'Waraqah Support', 'role': 'support'},
    {'id': 'u_reader', 'email': 'reader@waraqah.test', 'name': 'Reader', 'role': 'reader'},
  ]);
}
