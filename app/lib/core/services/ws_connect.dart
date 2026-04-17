import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:web_socket_channel/io.dart';

WebSocketChannel connectWs(Uri uri, {Map<String, String>? headers}) {
  return IOWebSocketChannel.connect(uri, headers: headers);
}
