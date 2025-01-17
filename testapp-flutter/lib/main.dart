import 'package:flutter/material.dart';
import 'package:pylons_sdk/pylons_sdk.dart';
import 'package:testapp_flutter/spinner.dart';

import 'game.dart';
import "spinner.dart";

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  PylonsWallet.setup(mode: PylonsMode.prod, host: 'testapp_flutter');

  runApp(const MyApp());
}

class MyApp extends StatefulWidget {
  const MyApp({Key? key}) : super(key: key);

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'BlockSlayer',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: const Spinner(child: Game(title: 'Welcome to BlockSlayer'))
    );
  }
}