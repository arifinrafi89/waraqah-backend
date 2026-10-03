// Shared pieces of the contract exporter: a sample request, the context that
// remembers earlier answers, and the file writer. See export_contract_test.dart.
import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';

/// Who sends the request on the real backend: guest, reader, or a staff role.
typedef As = String;

/// What earlier samples answered, so later samples can use real ids.
class Ctx {
  Ctx(this.dio);
  final Dio dio;
  final Map<String, dynamic> answers = {};

  /// The answer to the last sample with this "METHOD /path".
  dynamic of(String key) => answers[key];

  /// The id of the thing an earlier sample answered: the answer itself when it is an object,
  /// the first (or [last]) element when it is a list, or the element of the list stored under
  /// [inList] in an object. "unknown" when there is none, so a sample never crashes the run.
  String id(String key, {bool last = false, String? inList, String field = 'id'}) {
    Object? v = answers[key];
    if (inList != null && v is Map) v = v[inList];
    if (v is List) {
      if (v.isEmpty) return 'unknown';
      v = last ? v.last : v.first;
    }
    if (v is Map && v[field] != null) return '${v[field]}';
    return 'unknown';
  }

  /// The [field] of the first element of a list answer.
  String first(String key, String field) {
    final list = answers[key];
    if (list is List && list.isNotEmpty) return '${(list.first as Map)[field]}';
    throw StateError('no $field in the answer of $key');
  }
}

typedef Make<T> = T Function(Ctx ctx);

/// One request the exporter sends to the fake API and records.
class S {
  S.get(
    this.path, {
    this.query,
    this.as = 'guest',
    this.variant,
    this.sse = false,
  }) : method = 'GET',
       refusal = false,
       body = null;

  S.post(
    this.path, {
    this.body,
    this.as = 'reader',
    this.variant,
    this.refusal = false,
  }) : method = 'POST',
       query = null,
       sse = false;

  final String method;
  final String path;
  final Make<Map<String, dynamic>>? query;
  final Make<Object?>? body;
  final As as;
  final String? variant;
  final bool sse;

  /// The sample is a refusal: its golden answer is `null` on purpose.
  final bool refusal;

  String get key => '$method $path';

  String get fileName {
    final p = path.substring(1).replaceAll('/', '__');
    return '${method}__$p${variant == null ? '' : '__$variant'}.json';
  }
}

String encode(Object? v) => const JsonEncoder.withIndent('  ').convert(
  v,
  // ignore: avoid_dynamic_calls
);

Object? plain(Object? v) => jsonDecode(
  jsonEncode(v, toEncodable: (o) {
    if (o is DateTime) return o.toIso8601String();
    return o.toString();
  }),
);

/// Runs [body] with every timer firing at once, so the fake API's 900 ms
/// "loading" delay costs nothing.
Future<T> withoutDelays<T>(Future<T> Function() body) => runZoned(
  body,
  zoneSpecification: ZoneSpecification(
    createTimer: (self, parent, zone, d, f) =>
        parent.createTimer(zone, Duration.zero, f),
  ),
);

void writeJson(String path, Object? value) {
  final f = File(path)..createSync(recursive: true);
  f.writeAsStringSync('${encode(value)}\n');
}
