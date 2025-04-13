package net.thefoley.r38

import android.annotation.SuppressLint
import android.app.PendingIntent
import android.content.Intent
import android.content.IntentFilter
import android.nfc.NdefMessage
import android.nfc.NfcAdapter
import android.nfc.tech.NfcA
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.multiplatform.webview.web.WebView
import com.multiplatform.webview.web.rememberWebViewNavigator
import com.multiplatform.webview.web.rememberWebViewState
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

class MainActivity : ComponentActivity() {
    private var adapter: NfcAdapter? = null

    @OptIn(ExperimentalEncodingApi::class)
    @SuppressLint("SetJavaScriptEnabled")
    @Composable
    fun Content() {
        val state = rememberWebViewState("https://draftcu.be")
//    val state = rememberWebViewState("http://10.0.2.2:5173")
        val navigator = rememberWebViewNavigator()

        addOnNewIntentListener { intent ->
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

        WebView(state,
            modifier = Modifier.fillMaxSize(),
            navigator = navigator,
            onCreated = { webview -> webview.settings.javaScriptEnabled = true })
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
