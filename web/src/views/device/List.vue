<template>
  <el-card>
    <div class="toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜索名称 / 设备 ID"
        style="width: 260px"
        clearable
        @keyup.enter="reload"
        @clear="reload"
      />
      <el-button type="primary" @click="reload">查询</el-button>
      <el-button @click="reload">刷新</el-button>
    </div>

    <el-table :data="list" v-loading="loading" border>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="device_id" label="设备 ID" min-width="160" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.online ? 'success' : 'info'" size="small">
            {{ row.online ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="CPU" width="90">
        <template #default="{ row }">{{ row.online ? row.cpu_usage + '%' : '-' }}</template>
      </el-table-column>
      <el-table-column label="内存" width="150">
        <template #default="{ row }">
          {{ row.online ? fmtBytes(row.memory_used) + ' / ' + fmtBytes(row.memory_total) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="磁盘" width="150">
        <template #default="{ row }">
          {{ row.online ? fmtBytes(row.disk_used) + ' / ' + fmtBytes(row.disk_total) : '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="agent_version" label="版本" width="90" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" :disabled="!row.online" @click="openTerminal(row)">终端</el-button>
          <el-button link type="primary" :disabled="!row.online" @click="openFiles(row)">文件</el-button>
          <el-button link type="primary" @click="openConfig(row)">配置</el-button>
          <el-button link type="primary" :disabled="!row.online" @click="openTask(row)">任务</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      class="pager"
      layout="total, prev, pager, next"
      :total="total"
      :page-size="pageSize"
      :current-page="page"
      @current-change="onPage"
    />
  </el-card>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { searchDevices, deleteDevice } from '@/api/device'

const router = useRouter()
const loading = ref(false)
const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
let timer = null

async function load() {
  loading.value = true
  try {
    const res = await searchDevices({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value
    })
    list.value = res.data
    total.value = res.total || 0
  } catch (e) {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function onPage(p) {
  page.value = p
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`确认删除设备「${row.name}」？`, '提示', { type: 'warning' })
  await deleteDevice(row.id)
  ElMessage.success('已删除')
  load()
}

function openTerminal(row) {
  router.push(`/terminal/${row.device_id}`)
}

function openFiles(row) {
  router.push(`/files/${row.device_id}`)
}

function openConfig(row) {
  router.push(`/config/${row.device_id}`)
}

function openTask(row) {
  router.push(`/task/${row.device_id}`)
}

function fmtBytes(n) {
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return v.toFixed(1) + ' ' + units[i]
}

onMounted(() => {
  load()
  // 周期性刷新在线状态
  timer = setInterval(load, 5000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.pager {
  margin-top: 16px;
  justify-content: flex-end;
}
.file-name {
  margin-left: 12px;
  color: #909399;
  font-size: 13px;
}
</style>