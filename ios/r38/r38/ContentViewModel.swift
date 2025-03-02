import Foundation
import SwiftUI
import CoreNFC
import WebKit

class ContentViewModel: NSObject, ObservableObject, WKScriptMessageHandler , NFCNDEFReaderSessionDelegate {
	private var webView: WebView? = nil
	
	override init() {
		UIApplication.shared.isIdleTimerDisabled = true
	}
	
	deinit {
		UIApplication.shared.isIdleTimerDisabled = false
	}
	
	func registerWebView(webView: WebView) {
		self.webView = webView
	}
	
	private func startScan() {
		let nfcSession = NFCNDEFReaderSession(delegate: self, queue: nil, invalidateAfterFirstRead: false)
		nfcSession.begin()
	}
	
	func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
		startScan()
	}
	
	func readerSessionDidBecomeActive(_ session: NFCNDEFReaderSession) {}
	
	func readerSession(_ session: NFCNDEFReaderSession, didInvalidateWithError error: any Error) {
		if let nfcError = error as? NFCReaderError {
			switch nfcError.code {
			case NFCReaderError.Code.readerSessionInvalidationErrorUserCanceled:
				// fine
				break
			case NFCReaderError.Code.readerSessionInvalidationErrorSessionTimeout:
				startScan()
				break
			default:
				print(error)
				break
			}
		}
	}
	
	func readerSession(_ session: NFCNDEFReaderSession, didDetectNDEFs messages: [NFCNDEFMessage]) {
		if (self.webView == nil) {return}
		DispatchQueue.main.async {
			for message in messages {
				for record in message.records {
					self.webView!.emitNfcScan(payload: record.payload.base64EncodedString())
				}
			}
		}
	}
}
