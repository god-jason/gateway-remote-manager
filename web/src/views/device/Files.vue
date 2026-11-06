<template>
  <div class="files-page">
    <el-card>
      <div class="toolbar">
        <el-button link @click="goBack">
          <el-icon><ArrowLeft /></el-icon> 返回
        </el-button>
        <span class="device-label">设备：{{ deviceId }}</span>
        <span class="spacer" />
        <el-button :disabled="path === '/'" @click="goUp">上级</el-button>
        <el-input v-model="pathInput" class="path-input" placeholder="路径，如 /" @keyup.enter="navigate" />
        <el-button @click="navigate">前往</el-button>
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" @click="mkdirDialog = true">新建目录</el-button>
        <el-upload :auto-upload="false" :show-file-list="false" :on-change="onFileChange" :disabled="uploading">
          <el-button type="success" :loading="uploading" :disabled="uploading">上传文件</el-button>
        </el-upload>
      </div>

      <el-progress v-if="uploading" :percentage="progress" class="progress" />

      <el-table :data="list" v-loading="loading" border size="small">
        <el-table-column label="名称" min-width="280">
          <template #default="{ row }">
            <el-icon :class="row.is_dir ? 'dir-icon' : 'file-icon'">
              <Folder v-if="row.is_dir" />
              <Document v-else />
            </el-icon>
            <el-link type="primary" :underline="false" @click="openEntry(row)">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="110">
          <template #default="{ row }">{{ row.is_dir ? '-' : fmtBytes(row.size) }}</template>
        </el-table-column>
        <el-table-column prop="perm" label="权限" width="110" />
        <el-table-column label="修改时间" width="180">
          <template #default="{ row }">{{ fmtTime(row.modified_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.is_dir" link type="primary" @click="enterDir(row)">进入</el-button>
            <el-button v-else link type="primary" @click="download(row)">下载</el-button>
            <el-button v-if="!row.is_dir" link @click="openEdit(row)">编辑</el-button>
            <el-button link @click="openRename(row)">重命名</el-button>
            <el-button link type="danger" @click="removeEntry(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && list.length === 0" description="目录为空" />
    </el-card>

    <!-- 新建目录 -->
    <el-dialog v-model="mkdirDialog" title="新建目录" width="420px">
      <el-input v-model="newDirName" placeholder="目录名" @keyup.enter="doMkdir" />
      <template #footer>
        <el-button @click="mkdirDialog = false">取消</el-button>
        <el-button type="primary" @click="doMkdir">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重命名 -->
    <el-dialog v-model="renameDialog" title="重命名" width="420px">
      <el-input v-model="renameValue" placeholder="新名称" @keyup.enter="doRename" />
      <template #footer>
        <el-button @click="renameDialog = false">取消</el-button>
        <el-button type="primary" @click="doRename">确定</el-button>
      </template>
    </el-dialog>

    <!-- 编辑文本 -->
    <el-dialog v-model="editDialog" :title="editPath" width="760px" top="6vh">
      <div v-if="editTruncated" class="truncate-tip">文件过大，仅显示前 1MB（保存将覆盖为当前内容）</div>
      <el-input v-model="editContent" type="textarea" :rows="22" class="edit-area" />
      <template #footer>
        <el-button @click="editDialog = false">取消</el-button>
        <el-button type="primary" :loading="editSaving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fsList, fsMkdir, fsDelete, fsRename, fsRead, fsWrite } from '@/api/fs'
import { uploadFile, downloadFile } from '@/api/file'

const route = useRoute()
const router = useRouter()
const deviceId = route.params.deviceId

const loading = ref(false)
const list = ref([])
const path = ref('/')
const pathInput = ref('/')

const uploading = ref(false)
const progress = ref(0)

const mkdirDialog = ref(false)
const newDirName = ref('')
const renameDialog = ref(false)
const renameEntry = ref(null)
const renameValue = ref('')
const editDialog = ref(false)
const editPath = ref('')
const editContent = ref('')
const editTruncated = ref(false)
const editSaving = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fsList(deviceId, path.value)
    path.value = res.data.path || path.value
    pathInput.value = path.value
    list.value = res.data.entries || []
  } catch (e) {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

function navigate() {
  path.value = normalizePath(pathInput.value)
  load()
}

function goUp() {
  path.value = parentOf(path.value)
  pathInput.value = path.value
  load()
}

function enterDir(row) {
  path.value = row.path
  pathInput.value = row.path
  load()
}

function goBack() {
  router.replace('/device')
}

function openEntry(row) {
  if (row.is_dir) enterDir(row)
  else download(row)
}

function normalizePath(p) {
  if (!p) return '/'
  let s = p.trim()
  if (!s.startsWith('/')) s = '/' + s
  s = s.replace(/\/+/g, '/')
  return s === '' ? '/' : s
}

function parentOf(p) {
  if (p === '/' || !p) return '/'
  const parts = p.split('/').filter(Boolean)
  parts.pop()
  return parts.length === 0 ? '/' : '/' + parts.join('/')
}

function joinPath(dir, name) {
  return dir === '/' ? '/' + name : dir + '/' + name
}

function onFileChange(uploadFileObj) {
  const file = uploadFileObj.raw
  if (!file) return
  uploadToDevice(file)
}

async function uploadToDevice(file) {
  const remotePath = joinPath(path.value, file.name)
  uploading.value = true
  progress.value = 0
  try {
    await uploadFile(deviceId, file, remotePath, true, (p) => (progress.value = p))
    ElMessage.success('上传完成')
    load()
  } catch (e) {
    // 错误已提示
  } finally {
    uploading.value = false
  }
}

async function download(row) {
  try {
    await downloadFile(deviceId, row.path)
    ElMessage.success('下载完成')
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

function openRename(row) {
  renameEntry.value = row
  renameValue.value = row.name
  renameDialog.value = true
}

async function doRename() {
  if (!renameValue.value) {
    ElMessage.warning('请输入新名称')
    return
  }
  const src = renameEntry.value.path
  const dst = joinPath(parentOf(src), renameValue.value)
  try {
    await fsRename(deviceId, src, dst)
    ElMessage.success('已重命名')
    renameDialog.value = false
    load()
  } catch (e) {}
}

async function doMkdir() {
  if (!newDirName.value) {
    ElMessage.warning('请输入目录名')
    return
  }
  try {
    await fsMkdir(deviceId, joinPath(path.value, newDirName.value))
    ElMessage.success('已创建')
    mkdirDialog.value = false
    newDirName.value = ''
    load()
  } catch (e) {}
}

async function removeEntry(row) {
  await ElMessageBox.confirm(`确认删除「${row.name}」？${row.is_dir ? '（将递归删除目录）' : ''}`, '警告', { type: 'warning' })
  try {
    await fsDelete(deviceId, row.path)
    ElMessage.success('已删除')
    load()
  } catch (e) {}
}

async function openEdit(row) {
  try {
    const res = await fsRead(deviceId, row.path)
    editPath.value = row.path
    editContent.value = res.data.content || ''
    editTruncated.value = !!res.data.truncated
    editDialog.value = true
  } catch (e) {}
}

async function saveEdit() {
  editSaving.value = true
  try {
    await fsWrite(deviceId, editPath.value, editContent.value)
    ElMessage.success('已保存')
    editDialog.value = false
    load()
  } catch (e) {
  } finally {
    editSaving.value = false
  }
}

function fmtBytes(n) {
  if (n == null) return '0 B'
  if (n === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return v.toFixed(1) + ' ' + units[i]
}

function fmtTime(sec) {
  if (!sec) return '-'
  return new Date(sec * 1000).toLocaleString()
}

onMounted(load)
</script>

<style scoped>
.files-page {
  min-height: 100%;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.device-label {
  color: #909399;
  font-size: 13px;
}
.spacer {
  flex: 1;
}
.path-input {
  width: 260px;
}
.progress {
  margin-bottom: 12px;
}
.dir-icon {
  color: #e6a23c;
  margin-right: 6px;
  vertical-align: -2px;
}
.file-icon {
  color: #909399;
  margin-right: 6px;
  vertical-align: -2px;
}
.truncate-tip {
  color: #e6a23c;
  font-size: 12px;
  margin-bottom: 8px;
}
.edit-area :deep(textarea) {
  font-family: Menlo, Consolas, monospace;
  font-size: 13px;
}
</style>