<template>
  <div class="config-page">
    <el-card>
      <div class="toolbar">
        <el-button link @click="goBack">
          <el-icon><ArrowLeft /></el-icon> 返回
        </el-button>
        <span class="device-label">设备：{{ deviceId }}</span>
        <span class="spacer" />
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="openCreate">新增配置</el-button>
      </div>

      <el-table :data="list" v-loading="loading" border size="small">
        <el-table-column prop="config_key" label="配置 Key" min-width="200" />
        <el-table-column prop="config_type" label="类型" width="90">
          <template #default="{ row }"><el-tag size="small">{{ row.config_type }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="70" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTag(row)" size="small">{{ statusText(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="校验和" width="130">
          <template #default="{ row }">{{ short(row.checksum) }}</template>
        </el-table-column>
        <el-table-column label="更新时间" width="180">
          <template #default="{ row }">{{ fmtTime(row.updated) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && list.length === 0" description="暂无配置" />
    </el-card>

    <el-dialog v-model="dialog" :title="isNew ? '新增配置' : '编辑配置'" width="700px" top="6vh">
      <el-form label-width="90px" @submit.prevent>
        <el-form-item label="配置 Key">
          <el-input v-model="form.config_key" :disabled="!isNew" placeholder="如 app/app.yaml" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="form.content" type="textarea" :rows="18" class="editor" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存并下发</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { configList, configGet, configSet, configDelete } from '@/api/config'

const route = useRoute()
const router = useRouter()
const deviceId = route.params.deviceId

const loading = ref(false)
const list = ref([])
const dialog = ref(false)
const isNew = ref(true)
const saving = ref(false)
const form = ref({ config_key: '', content: '' })
const editType = ref('')

async function load() {
  loading.value = true
  try {
    const res = await configList(deviceId)
    list.value = res.data || []
  } catch (e) {
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isNew.value = true
  editType.value = ''
  form.value = { config_key: '', content: '' }
  dialog.value = true
}

async function openEdit(row) {
  try {
    const res = await configGet(deviceId, row.config_key)
    isNew.value = false
    editType.value = row.config_type
    form.value = { config_key: row.config_key, content: res.data.content || '' }
    dialog.value = true
  } catch (e) {}
}

async function save() {
  if (!form.value.config_key) {
    ElMessage.warning('请输入配置 Key')
    return
  }
  saving.value = true
  try {
    const type = editType.value || guessType(form.value.config_key)
    const res = await configSet(deviceId, form.value.config_key, type, form.value.content)
    ElMessage.success(res.data.synced ? '已保存并同步' : '已保存（设备离线，待下发）')
    dialog.value = false
    load()
  } catch (e) {
  } finally {
    saving.value = false
  }
}

async function remove(row) {
  await ElMessageBox.confirm(`确认删除配置「${row.config_key}」？`, '警告', { type: 'warning' })
  try {
    await configDelete(deviceId, row.config_key)
    ElMessage.success('已删除')
    load()
  } catch (e) {}
}

function goBack() {
  router.replace('/device')
}

function statusText(row) {
  if (row.synced) return '已同步'
  if (row.status === 'pending') return '待下发'
  if (row.status === 'failed') return '下发失败'
  return row.status || '-'
}

function statusTag(row) {
  if (row.synced) return 'success'
  if (row.status === 'failed') return 'danger'
  if (row.status === 'pending') return 'info'
  return 'info'
}

function short(s) {
  return s ? s.slice(0, 12) : '-'
}

function fmtTime(v) {
  if (!v) return '-'
  const d = new Date(v)
  return isNaN(d.getTime()) ? String(v) : d.toLocaleString()
}

function guessType(key) {
  if (/\.ya?ml$/.test(key)) return 'yaml'
  if (/\.json$/.test(key)) return 'json'
  if (/\.toml$/.test(key)) return 'toml'
  if (/\.ini$/.test(key)) return 'ini'
  return 'text'
}

onMounted(load)
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
.editor :deep(textarea) {
  font-family: Menlo, Consolas, monospace;
  font-size: 13px;
}
</style>