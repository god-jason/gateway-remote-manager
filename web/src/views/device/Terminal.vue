<template>
  <div class="terminal-page">
    <div class="terminal-bar">
      <el-button link @click="goBack">
        <el-icon><ArrowLeft /></el-icon> 返回
      </el-button>
      <span class="title">远程终端 · {{ deviceId }}</span>
      <span class="state" :class="{ online: connected }">{{ connected ? '已连接' : '未连接' }}</span>
    </div>
    <div ref="terminalEl" class="terminal-container"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Cookies from 'js-cookie'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const route = useRoute()
const router = useRouter()
const deviceId = route.params.deviceId

const terminalEl = ref(null)
const connected = ref(false)

let term = null
let ws = null
let fitAddon = null

function wsProtocol() {
  return location.protocol === 'https:' ? 'wss:' : 'ws:'
}

function openTerminal() {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Consolas, "Courier New", monospace',
    theme: { background: '#1e1e1e' },
    convertEol: true
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalEl.value)
  fitAddon.fit()

  const token = Cookies.get('token')
  const url = `${wsProtocol()}//${location.host}/api/terminal/ws?token=${token}&device_id=${deviceId}`
  ws = new WebSocket(url)

  ws.onopen = () => {
    connected.value = true
    ws.send(JSON.stringify({ type: 'open', device_id: deviceId, shell: '/bin/bash', cols: term.cols, rows: term.rows }))
  }
  ws.onmessage = (ev) => {
    let data = ev.data
    if (typeof data === 'string') {
      try {
        const m = JSON.parse(data)
        if (m.type === 'error') {
          term.write(`\r\n\x1b[31m[错误] ${m.message}\x1b[0m\r\n`)
          return
        }
      } catch (e) {
        // 非 JSON，视为普通终端输出
      }
    }
    term.write(data)
  }
  ws.onclose = () => {
    connected.value = false
    term.write('\r\n\x1b[33m[连接已关闭]\x1b[0m\r\n')
  }
  ws.onerror = () => {}

  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })
  term.onResize(({ cols, rows }) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
    fitAddon.fit()
  })
  window.addEventListener('resize', onResize)
}

function onResize() {
  if (fitAddon) fitAddon.fit()
}

function goBack() {
  router.replace('/device')
}

onMounted(openTerminal)

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  if (ws) ws.close()
  if (term) term.dispose()
})
</script>

<style scoped>
.terminal-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}
.terminal-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #2b2b2b;
  color: #e6e6e6;
}
.title {
  font-size: 14px;
}
.state {
  margin-left: auto;
  font-size: 12px;
  color: #f56c6c;
}
.state.online {
  color: #67c23a;
}
.terminal-container {
  flex: 1;
  padding: 8px;
}
</style>