package net.thefoley.r38

import android.annotation.SuppressLint
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.kevinnzou.web.WebView
import com.kevinnzou.web.rememberWebViewState

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        android.webkit.WebView.setWebContentsDebuggingEnabled(true);
        setContent {
            Content()
        }
    }
}

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun Content() {
    val state = rememberWebViewState("https://draft.thefoley.net")
//    val state = rememberWebViewState("http://10.0.2.2:5173")
    WebView(state, modifier = Modifier.fillMaxSize(),
        onCreated = { it.settings.javaScriptEnabled = true })
}