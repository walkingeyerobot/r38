import CoreNFC

class NFCReaderDelegate: NSObject, NFCNDEFReaderSessionDelegate {
	private var webView: WebView
	init(webView: WebView) {
		self.webView = webView
	}

	func readerSession(_ session: NFCNDEFReaderSession, didInvalidateWithError error: any Error) {
		
	}
	
	func readerSession(_ session: NFCNDEFReaderSession, didDetectNDEFs messages: [NFCNDEFMessage]) {
		DispatchQueue.main.async {
			for message in messages {
				for record in message.records {
					self.webView.emitNfcScan(payload: record.payload.base64EncodedString())
				}
			}
		}
	}
}
