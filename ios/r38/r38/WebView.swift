import SwiftUI
import WebKit

struct WebView: UIViewRepresentable {
	let url: URL
	private var webView: WKWebView
	
	init(url: URL) {
		self.url = url
		self.webView = WKWebView()
	}
	
	func makeUIView(context: Context) -> WKWebView {
		return self.webView
	}
	
	func updateUIView(_ webView: WKWebView, context: Context) {
		let request = URLRequest(url: url)
		webView.load(request)
	}
	
	func back() {
		self.webView.goBack()
	}
	
	func refresh() {
		self.webView.reload()
	}
	
	func forward() {
		self.webView.goForward()
	}
	
	func emitNfcScan(payload: String) {
		self.webView.evaluateJavaScript(
			"document.body.dispatchEvent(new CustomEvent('rfidScan', {detail: '\(payload)}))")
	}
}

#Preview {
	WebView(url: URL(string: "https://draft.thefoley.net")!)
}
