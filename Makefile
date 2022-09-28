mobile: android ios

ensure-path:
	@mkdir -p ./build/mobile/android
	@mkdir -p ./build/mobile/ios

android: ensure-path
	gomobile bind -a -ldflags '-s -w' -trimpath -target=android -o ./build/mobile/android/httpssni.aar github.com/blacktear23/httpssni/httpssni

ios: ensure-path
	gomobile bind -a -ldflags '-s -w' -trimpath -target=ios -o ./build/mobile/ios/httpssni.xcframework github.com/blacktear23/httpssni/httpssni
