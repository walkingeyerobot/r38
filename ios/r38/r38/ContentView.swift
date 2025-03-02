import SwiftUI
import CoreNFC

struct ContentView: View {
	private var webView: WebView
	@State private var viewModel: ContentViewModel
	
	init() {
		let viewModel = ContentViewModel()
		self.viewModel = viewModel
		self.webView = WebView(url: URL(string: "https://draft.thefoley.net")!,
		//	self.webView = WebView(url: URL(string: "http://localhost:5173")!,
							   webViewDelegate: viewModel)
		viewModel.registerWebView(webView: webView)
	}
	
	var body: some View {
	  VStack {
		webView
		
		HStack {
		  Button(action: {
			self.webView.back()
		  }){
			Image(systemName: "arrowtriangle.left.fill")
							  .font(.title)
							  .foregroundStyle(.black)
							  .padding()
		  }
		  Spacer()
		  Button(action: {
			self.webView.refresh()
		  }){
			Image(systemName: "arrow.clockwise.circle")
				  .font(.title)
				  .foregroundStyle(.black)
				  .padding()
		  }
		  Spacer()
		  Button(action: {
			self.webView.forward()
		  }){
			Image(systemName: "arrowtriangle.right.fill")
				  .font(.title)
				  .foregroundStyle(.black)
				  .padding()
		  }
		}
	  }
	  .preferredColorScheme(.light)
	}
}

#Preview {
    ContentView()
}
