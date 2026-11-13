import 'package:flutter/material.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

import 'data/rtc_bridge.dart';

/// WebRTC 视频渲染区：远端全屏，本地预览悬浮在右上角。
class RtcVideoPanel extends StatelessWidget {
  const RtcVideoPanel({super.key, required this.bridge});

  final RtcBridge bridge;

  @override
  Widget build(BuildContext context) {
    final remote = bridge.remoteRenderer;
    final local = bridge.localRenderer;
    return ClipRRect(
      borderRadius: BorderRadius.circular(20),
      child: ColoredBox(
        color: Colors.black,
        child: Stack(
          fit: StackFit.expand,
          children: [
            if (remote != null)
              RTCVideoView(
                remote,
                objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitCover,
              )
            else
              const Center(
                child: Text(
                  '等待对方画面…',
                  style: TextStyle(color: Colors.white70),
                ),
              ),
            if (local != null)
              Positioned(
                right: 12,
                top: 12,
                width: 108,
                height: 152,
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(12),
                  child: RTCVideoView(
                    local,
                    mirror: true,
                    objectFit:
                        RTCVideoViewObjectFit.RTCVideoViewObjectFitCover,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
