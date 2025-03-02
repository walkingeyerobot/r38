import SwiftUI
import WebKit

struct WebView: UIViewRepresentable {
	let url: URL
	private var webView: WKWebView
	
	init(url: URL, webViewDelegate: WKScriptMessageHandler) {
		self.url = url
		self.webView = WKWebView()
		self.webView.allowsBackForwardNavigationGestures = true
		self.webView.allowsLinkPreview = true
		self.webView.configuration.userContentController.add(webViewDelegate, name: "scanner")
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
			"document.body.dispatchEvent(new CustomEvent('rfidScan', {detail: '\(payload)'}))")
	}
}
