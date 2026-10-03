// Contract exporter. It runs every sample against the fake API and writes one golden file per
// request to build/contract/, plus the seed files in build/seed/. The backend's
// scripts/export-contract.sh copies this file into the frontend's test/tool/, runs it, and copies
// the output back (BACKEND_PLAN.md section 14).
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:waraqah/app/fake_api_routes.dart';
import 'package:waraqah/app/fake_stores.dart';

import 'harness.dart';
import 'samples_reads.dart';
import 'samples_writes.dart';
import 'seeds.dart';

void main() {
  test('export contract goldens and seeds', () async {
    await withoutDelays(() async {
      final dio = Dio()..interceptors.add(FakeApiRoutes.interceptor(FakeStores()));
      final ctx = Ctx(dio);
      final out = Directory('build/contract');
      if (out.existsSync()) out.deleteSync(recursive: true);
      out.createSync(recursive: true);

      final index = <Map<String, Object?>>[];
      final problems = <String>[];
      final names = <String>{};
      for (final s in [...readSamples(), ...writeSamples()]) {
        if (!names.add(s.fileName)) problems.add('${s.key}: two samples write ${s.fileName}; give one a variant');
        final query = s.query?.call(ctx) ?? const <String, dynamic>{};
        final body = s.body?.call(ctx);
        final request = {
          'method': s.method,
          'path': s.path,
          'query': query,
          'body': body,
          'as': s.as,
        };
        Object? response;
        if (!s.sse) {
          try {
            final r = await dio.fetch<dynamic>(
              RequestOptions(
                path: s.path,
                method: s.method,
                queryParameters: query,
                data: body,
              ),
            );
            response = plain(r.data);
            if (s.variant == null) ctx.answers[s.key] = response;
          } on DioException catch (e) {
            problems.add('${s.key}: ${e.response?.statusCode ?? e.message}');
            continue;
          } on Object catch (e) {
            problems.add('${s.key}: $e');
            continue;
          }
          if (response == null && s.variant == null && !s.refusal) {
            problems.add('${s.key}: answered null (check the sample)');
          }
        }
        writeJson('${out.path}/${s.fileName}', {
          'request': request,
          'response': response,
          if (s.sse) 'sse': true,
        });
        index.add({'file': s.fileName, ...request, if (s.sse) 'sse': true});
      }
      writeJson('${out.path}/_index.json', index);
      File('${out.path}/_problems.txt').writeAsStringSync(problems.join('\n'));
      // When the goldens were made: Home's Season and similar answers depend on the day.
      writeJson('${out.path}/_meta.json', {'exportedAt': DateTime.now().toIso8601String()});

      await writeSeeds(Directory('build/seed'));
      stdout.writeln('exported ${index.length} goldens, ${problems.length} problems');
      for (final p in problems) {
        stdout.writeln('  PROBLEM $p');
      }
    });
  }, timeout: const Timeout(Duration(minutes: 15)));
}
