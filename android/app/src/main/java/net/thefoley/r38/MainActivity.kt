package net.thefoley.r38

import android.annotation.SuppressLint
import android.app.PendingIntent
import android.content.Intent
import android.content.IntentFilter
import android.nfc.NdefMessage
import android.nfc.NdefRecord
import android.nfc.NfcAdapter
import android.nfc.Tag
import android.nfc.tech.Ndef
import android.nfc.tech.NfcA
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.multiplatform.webview.jsbridge.IJsMessageHandler
import com.multiplatform.webview.jsbridge.JsMessage
import com.multiplatform.webview.jsbridge.rememberWebViewJsBridge
import com.multiplatform.webview.web.WebView
import com.multiplatform.webview.web.WebViewNavigator
import com.multiplatform.webview.web.rememberWebViewNavigator
import com.multiplatform.webview.web.rememberWebViewState
import org.json.JSONObject
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

class MainActivity : ComponentActivity() {
    private var adapter: NfcAdapter? = null
    private var cardToWrite: String? = null

    @OptIn(ExperimentalEncodingApi::class)
    @SuppressLint("SetJavaScriptEnabled")
    @Composable
    fun Content() {
        val state = rememberWebViewState("https://draftcu.be")
//    val state = rememberWebViewState("http://10.0.2.2:5173")
        val navigator = rememberWebViewNavigator()
        val jsBridge = rememberWebViewJsBridge(navigator)

        addOnNewIntentListener { intent ->
            if (!this.cardToWrite.isNullOrBlank()) {
                val tag = intent.getParcelableExtra(
                    NfcAdapter.EXTRA_TAG, Tag::class.java
                )
                if (tag != null) {
                    Ndef.get(tag).writeNdefMessage(
                        NdefMessage(
                            NdefRecord.createTextRecord(null, this.cardToWrite)
                        )
                    )
                }
            } else {
                val messages = intent.getParcelableArrayExtra(
                    NfcAdapter.EXTRA_NDEF_MESSAGES, NdefMessage::class.java
                )
                if (messages != null) {
                    for (record in messages.flatMap { it.records.asIterable() }) {
                        val payload = Base64.encode(record.payload)
                        navigator.evaluateJavaScript(
                            "document.body.dispatchEvent(new CustomEvent('rfidScan', {detail: '$payload'}))"
                        )
                    }
                }
            }
        }

        jsBridge.register(object : IJsMessageHandler {
            override fun handle(
                message: JsMessage,
                navigator: WebViewNavigator?,
                callback: (String) -> Unit
            ) {
                val json = JSONObject(message.params)
                this@MainActivity.cardToWrite = json.optString("card")
            }

            override fun methodName() = "setCard"
        })

        WebView(state,
            modifier = Modifier.fillMaxSize(),
            navigator = navigator,
            onCreated = { webview -> webview.settings.javaScriptEnabled = true },
            webViewJsBridge = jsBridge,
        )
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        this.adapter = NfcAdapter.getDefaultAdapter(this)
        android.webkit.WebView.setWebContentsDebuggingEnabled(true);
        setContent {
            Content()
        }
    }

    override fun onPause() {
        super.onPause()
        adapter?.disableForegroundDispatch(this)
    }

    override fun onResume() {
        super.onResume()
        val intent = Intent(this, javaClass).apply {
            addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
        }
        val pendingIntent: PendingIntent = PendingIntent.getActivity(
            this, 0, intent, PendingIntent.FLAG_MUTABLE
        )
        val ndef = IntentFilter(NfcAdapter.ACTION_NDEF_DISCOVERED).apply {
            try {
                addDataType("*/*")
            } catch (e: IntentFilter.MalformedMimeTypeException) {
                throw RuntimeException("fail", e)
            }
        }
        val intentFiltersArray = arrayOf(ndef)
        val techListsArray = arrayOf(arrayOf<String>(NfcA::class.java.name))
        adapter?.enableForegroundDispatch(this, pendingIntent, intentFiltersArray, techListsArray)
    }
}
