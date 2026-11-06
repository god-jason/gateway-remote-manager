<template>
  <div class="task-page">
    <el-card>
      <div class="toolbar">
        <el-button link @click="goBack">
          <el-icon><ArrowLeft /></el-icon> 返回
        </el-button>
        <span class="device-label">设备：{{ deviceId }}</span>
        <span class="spacer" />
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="openCreate">执行命令</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border size="small">
        <el-table-column label="任务 ID" width="120">
          <template #default="{ row }">{{ short(row.task_id) }}</template>
        </el-table-column>
        <el-table-column prop="command" label="命令" min-width="240" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="exit_code" label="退出码" width="80" />
        <el-table-column label="超时(s)" width="80">
          <template #default="{ row }">{{ row.timeout_seconds }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.created) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <el-button v-if="isRunning(row.status)" link type="danger" @click="cancel(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && list.length === 0" description="暂无任务" />
    </el-card>

    <!-- 执行命令 -->
    <el-dialog v-model="createDialog" title="执行命令" width="620px">
      <el-form label-width="90px" @submit.prevent>
        <el-form-item label="命令">
          <el-input v-model="form.command" type="textarea" :rows="6" placeholder="如 systemctl restart xxx" class="mono" />
        </el-form-item>
        <el-form-item label="超时(秒)">
          <el-input-number v-model="form.timeout_seconds" :min="1" :max="3600" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="create">执行</el-button>
      </template>
    </el-dialog>

    <!-- 任务详情 -->
    <el-dialog v-model="detailDialog" :title="'任务详情 · ' + short(detail.task_id)" width="680px" top="6vh">
      <div class="detail-meta">
        <el-tag :type="statusTag(detail.status)" size="small">{{ statusText(detail.status) }}</el-tag>
        <span v-if="detail.exit_code !== 0 && detail.exit_code != null">退出码：{{ detail.exit_code }}</span>
      </div>
      <div class="detail-cmd">命令：{{ detail.command }}</div>
      <el-input v-model="detail.output" type="textarea" :rows="16" readonly class="mono" />
      <template #footer>
        <el-button @click="detailDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { taskCreate, taskList, taskGet, taskCancel } from '@/api/task'

const route = useRoute()
const router = useRouter()
const deviceId = route.params.deviceId

const loading = ref(false)
const list = ref([])
const createDialog = ref(false)
const creating = ref(false)
const form = ref({ command: '', timeout_seconds: 60 })

const detailDialog = ref(false)
const detail = ref({})

let timer = null

async function load() {
  loading.value = true
  try {
    const res = await taskList(deviceId)
    list.value = res.data || []
  } catch (e) {
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = { command: '', timeout_seconds: 60 }
  createDialog.value = true
}

async function create() {
  if (!form.value.command) {
    ElMessage.warning('请输入命令')
    return
  }
  creating.value = true
  try {
    await taskCreate(deviceId, form.value.command, form.value.timeout_seconds)
    ElMessage.success('任务已下发')
    createDialog.value = false
    load()
  } catch (e) {
  } finally {
    creating.value = false
  }
}

async function openDetail(row) {
  try {
    const res = await taskGet(deviceId, row.task_id)
    detail.value = res.data || {}
    detailDialog.value = true
  } catch (e) {}
}

async function cancel(row) {
  await ElMessageBox.confirm(`确认取消任务「${short(row.task_id)}」？`, '提示', { type: 'warning' })
  try {
    await taskCancel(deviceId, row.task_id)
    ElMessage.success('已取消')
    load()
  } catch (e) {}
}

function goBack() {
  router.replace('/device')
}

function short(s) {
  return s ? s.slice(0, 8) : '-'
}

function statusText(s) {
  const map = {
    pending: '待下发',
    dispatched: '执行中',
    success: '成功',
    failed: '失败',
    timeout: '超时',
    cancelled: '已取消',
    error: '下发失败'
  }
  return map[s] || s || '-'
}

function statusTag(s) {
  switch (s) {
    case 'success':
      return 'success'
    case 'failed':
    case 'error':
      return 'danger'
    case 'timeout':
      return 'warning'
    case 'dispatched':
      return 'primary'
    default:
      return 'info'
  }
}

function isRunning(s) {
  return s === 'pending' || s === 'dispatched'
}

function fmtTime(v) {
  if (!v) return '-'
  const d = new Date(v)
  return isNaN(d.getTime()) ? String(v) : d.toLocaleString()
}

onMounted(() => {
  load()
  timer = setInterval(load, 3000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.device-label {
  color: #909399;
  font-size: 13px;
}
.spacer {
  flex: 1;
}
.mono :deep(textarea) {
  font-family: Menlo, Consolas, monospace;
  font-size: 13px;
}
.detail-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  color: #606266;
}
.detail-cmd {
  margin-bottom: 8px;
  color: #606266;
  font-size: 13px;
}
</style>