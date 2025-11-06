<template>
	<div
		class="flex flex-col h-screen bg-white"
		@contextmenu.prevent
	>
		<!-- 顶部标题栏 -->
		<div class="p-4 border-b flex items-center justify-between bg-gray-50 relative">
			<!-- 移动端左栏展开按钮 -->
			<NButton
				v-if="isMobile"
				@click="togglePanel"
				type="primary"
				ghost
				size="small"
				class="flex items-center gap-1 text-blue-600 hover:bg-blue-100 transition-colors border border-blue-200"
			>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
				</svg>
			</NButton>

			<NButton
				@click="goBackToHome"
				type="primary"
				ghost
				size="small"
				class="flex items-center gap-1 text-blue-600 hover:bg-blue-100 transition-colors border border-blue-200"
			>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
				{{ t('bookmarkManager.back') }}
			</NButton>

			<h1 class="text-xl font-bold text-gray-800 flex-1 text-center">{{ t('bookmarkManager.management') }}</h1>

			<NButton
				@click="createNewBookmark"
				type="primary"
				ghost
				size="small"
				class="flex items-center gap-1 text-purple-600 hover:bg-purple-100 transition-colors border border-purple-200"
			>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				{{ t('bookmarkManager.create') }}
			</NButton>

			<NButton
				@click="triggerImportBookmarks"
				type="success"
				ghost
				size="small"
				:loading="uploadLoading"
				class="flex items-center gap-1 text-green-600 hover:bg-green-100 transition-colors border border-green-200"
			>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
				{{ t('bookmarkManager.importBookmarks') }}
			</NButton>

			<!-- 隐藏的文件输入框 -->
			<input
				ref="fileInput"
				type="file"
				accept=".html"
				style="display: none;"
				@change="handleFileChange"
			/>
		</div>

		<!-- 主内容区域 -->
		<div class="flex flex-1 overflow-hidden">
			<!-- 遮罩层：移动端左栏打开时 -->
			<div
				v-if="isMobile && showLeftPanel"
				class="fixed inset-0 bg-black bg-opacity-30 z-40"
				@click="collapsePanel"
			></div>

			<!-- 左侧书签树 -->
			<div
				v-show="showLeftPanel"
				:class="[
    isMobile ? 'fixed top-0 left-0 h-full bg-white z-50 rounded-r-lg shadow-lg overflow-auto transition-all duration-300 ease-in-out' : 'h-full bg-white border-r border-gray-200 overflow-auto',
    isMobile && isPanelExpanded ? 'w-3/4' : isMobile ? 'w-12' : ''
  ]"
				:style="{ width: !isMobile ? leftPanelWidth + 'px' : '' }"
			>
				<n-tree
					:data="bookmarkTree"
					:default-expanded-keys="defaultExpandedKeys"
					block-line
					@update:selected-keys="handleSelect"
					:render-label="renderTreeLabel"
					:render-expand-icon="renderExpandIcon"
					ref="treeRef"
				/>
			</div>

			<!-- 可拖动分割线（桌面端） -->
			<div
				v-if="!isMobile"
				class="w-1 bg-gray-200 cursor-col-resize hover:bg-blue-300 flex items-center justify-center"
				@mousedown="startResize"
				:style="{ height: '100%', userSelect: 'none' }"
			>
				<div class="w-px h-12 bg-gray-400"></div>
			</div>

			<!-- 右侧书签列表 -->
		<div class="flex-1 flex flex-col overflow-hidden">
			<div class="p-2 border-b flex flex-col">
				<!-- 面包屑导航 -->
				<div class="flex items-center text-sm mb-2 text-gray-600">
					<span v-for="(crumb, index) in currentPath" :key="index"
					      class="cursor-pointer hover:text-blue-600"
					      @click="handleBreadcrumbClick(crumb.id)">
						{{ crumb.name }}
						<span v-if="index < currentPath.length - 1" class="mx-1">/</span>
					</span>
				</div>
				<n-input
					v-model:value="searchQuery"
					:placeholder="t('bookmarkManager.searchPlaceholder')"
					clearable
					@input="handleSearch"
				/>
			</div>

				<!-- 书签列表 -->
				<div class="flex-1 p-4 relative overflow-auto">
					<div class="grid grid-cols-1 gap-2">
						<div v-if="filteredBookmarks.length === 0" class="text-center py-8 text-gray-400">
							{{ t('bookmarkManager.noData') }}
						</div>

						<div
			v-for="item in filteredBookmarks"
			:key="item.id"
			class="flex items-center justify-between border p-2 rounded hover:bg-gray-50 cursor-pointer"
			@contextmenu.prevent="openContextMenu($event, item)"
			@click="item.isFolder ? selectedFolder = String(item.id) : openBookmark(item)"
			@dblclick="item.isFolder && enterFolder(String(item.id))"
		>
			<div class="flex items-center space-x-2">
				<span v-if="item.isFolder" class="text-blue-600">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 inline mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
					</svg>
				</span>
				<span class="font-medium text-slate-700">{{ item.title }}</span>
				<span v-if="!item.isFolder" class="text-slate-400 text-sm truncate max-w-[200px] whitespace-nowrap">{{ item.url }}</span>
			</div>
		</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 编辑书签对话框 -->
		<div v-if="isEditDialogOpen" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
			<div class="bg-white p-6 rounded-lg w-96">
				<h3 class="text-xl font-bold mb-4">{{ isCreateMode ? t('bookmarkManager.createBookmark') : t('bookmarkManager.editBookmark') }}</h3>
				<div v-if="isCreateMode" class="mb-4">
					<label class="block mb-2">{{ t('bookmarkManager.type') }}</label>
					<select
						v-model="bookmarkType"
						class="w-full px-3 py-2 border border-gray-300 rounded-md"
					>
						<option value="bookmark">{{ t('bookmarkManager.bookmark') }}</option>
						<option value="folder">{{ t('bookmarkManager.folder') }}</option>
					</select>
				</div>
				<div class="mb-4">
					<label class="block mb-2">{{ t('bookmarkManager.title') }}</label>
					<input
						v-model="currentEditBookmark.title"
						class="w-full px-3 py-2 border border-gray-300 rounded-md"
						:placeholder="t('bookmarkManager.title')"
					/>
				</div>
				<div v-if="(!isCreateMode && !currentBookmark?.isFolder) || (isCreateMode && bookmarkType === 'bookmark')" class="mb-4">
					<label class="block mb-2">{{ t('bookmarkManager.url') }}</label>
					<input
						v-model="currentEditBookmark.url"
						class="w-full px-3 py-2 border border-gray-300 rounded-md"
						:placeholder="t('bookmarkManager.enterUrl')"
					/>
				</div>
				<div class="mb-4">
					<label class="block mb-2">{{ t('bookmarkManager.parentFolder') }}</label>
					<select v-model="currentEditBookmark.folderId" class="w-full px-3 py-2 border border-gray-300 rounded-md">
						<option v-for="folder in allFolders" :key="folder.value" :value="folder.value">{{ folder.value === '0' ? t('bookmarkManager.rootDirectory') : folder.label }}</option>
					</select>
				</div>
				<div class="flex justify-end gap-2">
					<button @click="closeEditDialog" class="px-4 py-2 border border-gray-300 rounded-md">{{ t('bookmarkManager.cancel') }}</button>
					<button @click="saveBookmarkChanges" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors shadow-md">{{ t('bookmarkManager.confirm') }}</button>
				</div>
			</div>
		</div>

		<!-- 右键菜单 -->
		<div
			v-if="isContextMenuOpen"
			:style="contextMenuStyle"
			class="fixed bg-white text-gray-700 shadow-lg rounded-md py-1 z-50 w-40 context-menu border border-gray-200"
			@contextmenu.prevent.stop
		>
			<div v-if="!currentBookmark?.isFolder" @click="handleEditBookmark" class="px-4 py-2 hover:bg-gray-100 cursor-pointer">{{ t('bookmarkManager.edit') }}</div>
			<div @click="handleDeleteBookmark" class="px-4 py-2 hover:bg-gray-100 cursor-pointer">{{ t('bookmarkManager.delete') }}</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, h } from 'vue'
import FolderOpenIcon from '@/assets/svg-icons/icon_folder_open.svg'
// 不再直接导入SVG文件，使用内联方式
import { NTree, NInput, NButton, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import { addMultiple as addMultipleBookmarks, add as addBookmark, getList as getBookmarksList, deletes, update as updateBookmark } from '@/api/panel/bookmark'
import { t } from '@/locales'
import { dialog } from '@/utils/request/apiMessage'
import { ss } from '@/utils/storage/local'


// 组件顶部（import 之后）添加核心接口
interface TreeOption {
	key: string; // 节点唯一标识
	label: string; // 显示文本
	isLeaf: boolean; // 是否为叶子节点（书签）
	isFolder: boolean; // 是否为文件夹（核心标识）
	bookmark?: Omit<Bookmark, 'folderId'>; // 书签信息（非文件夹节点）
	children: TreeOption[]; // 子节点（必须初始化为数组）
	rawNode: any; // 后端原始数据
	disabledExpand: boolean; // 关键：是否禁用折叠（控制图标显示）
}
// 缓存键名定义
const  BOOKMARKS_FULL_CACHE_KEY= 'bookmarksTreeCache' // 只包含文件夹的树结构缓存
const BOOKMARKS_CACHE_KEY = 'bookmarksFullCache' // 完整书签数据缓存

const router = useRouter()
const ms = useMessage()
const isMobile = ref(false)      // 是否移动端
const showLeftPanel = ref(true)  // 左栏是否显示
// 左侧面板宽度
const leftPanelWidth = ref(256) // 默认256px
const isResizing = ref(false)
// 只展开第一级节点的键
const defaultExpandedKeys = ref<string[]>([])

// 导入相关
const fileInput = ref<HTMLInputElement>()
const uploadLoading = ref(false)
const jsonData = ref<string | null>(null)
const importWarning = ref<string[]>([])
const isPanelExpanded = ref(false)
// 返回首页
function goBackToHome() {
  router.push('/')
}


function togglePanel() {
	showLeftPanel.value = true
	isPanelExpanded.value = true
}

function collapsePanel() {
	isPanelExpanded.value = false
	setTimeout(() => {
		showLeftPanel.value = false
	}, 300) // 等待动画完成再隐藏
}

// 开始调整大小
function startResize(e: MouseEvent) {
  isResizing.value = true
  // 防止文本选择
  e.preventDefault()
}

// 处理鼠标移动
function handleMouseMove(e: MouseEvent) {
  if (!isResizing.value) return

  // 获取主内容区域的位置
  const container = document.querySelector('.flex-1.overflow-hidden') as HTMLElement
  if (!container) return

  const containerRect = container.getBoundingClientRect()
  const newWidth = e.clientX - containerRect.left

  // 设置最小宽度和最大宽度限制
  if (newWidth > 150 && newWidth < containerRect.width - 200) {
    leftPanelWidth.value = newWidth
  }
}

// 结束调整大小
function stopResize() {
  isResizing.value = false
}


interface Bookmark {
	id: number
	title: string
	url: string
	folderId?: string | number
	isFolder?: boolean
	parentId?: number
	createTime?: string
	updateTime?: string
}


const bookmarkTree = ref<any[]>([])

// 当前选中的文件夹
const selectedFolder = ref<string>('')

// 新增：响应式的完整数据变量
const fullData = ref<any[]>([])

// 计算属性 - 所有书签和文件夹
const allItems = computed<(Bookmark & { isFolder?: boolean })[]>(() => {
	const items: (Bookmark & { isFolder?: boolean })[] = []

	// 获取完整数据，优先使用响应式的fullData变量
	const data = fullData.value.length > 0 ? fullData.value : bookmarkTree.value

	// 遍历所有文件夹和书签
	function traverseItems(nodes: any[], folderId: string) {
		for (const node of nodes) {
			// 如果是文件夹，添加到列表并继续遍历
			if (node.isFolder || (!node.isLeaf && !node.bookmark?.url)) {
				items.push({
					id: Number(node.key),
					title: node.label,
					url: '',
					folderId: String(folderId),
					isFolder: true
				})
			}
			// 如果是书签节点，添加到列表
			if ((node.isLeaf && node.bookmark) || (!node.isFolder && !node.isLeaf && node.id !== undefined)) {
				const bookmarkData = node.bookmark || node
				items.push({
					...bookmarkData,
					folderId: String(folderId),
					isFolder: false
				})
			}
			// 继续遍历子节点
			if (node.children && node.children.length > 0) {
				traverseItems(node.children, String(node.key))
			}
		}
	}

	// 开始遍历完整数据
	traverseItems(data, '0')
	return items
})

// 搜索
const searchQuery = ref('')
const filteredBookmarks = computed(() => {
	// 如果选中了具体书签，直接返回该书签
	if (selectedBookmarkId.value) {
		const bookmark = allItems.value.find(b => String(b.id) === selectedBookmarkId.value);
		return bookmark ? [bookmark] : [];
	}

	// 先获取所有项目（书签和文件夹）
	let items = allItems.value
	if (selectedFolder.value) {
		const targetFolderId = String(selectedFolder.value)
		items = items.filter(item => {
			// 直接比较字符串形式的folderId
			return String(item.folderId) === targetFolderId
		})
	}

	// 应用搜索过滤
	if (searchQuery.value.trim()) {
		const query = searchQuery.value.toLowerCase()
		items = items.filter(item =>
			item.title.toLowerCase().includes(query) ||
			(item.url && item.url.toLowerCase().includes(query))
		)
	}

	return items
})

// 当前选中的节点键引用
const selectedKeysRef = ref<(string | number)[]>([]);

// 当前选中的书签ID（用于显示单个书签）
const selectedBookmarkId = ref<string>('')

// 查找节点并构建路径
function findNodePath(nodes: any[], targetId: string, currentPath: {id: string, name: string}[] = []): {id: string, name: string}[] | null {
	for (const node of nodes) {
		// 先将当前节点添加到路径中
		const newPath = [...currentPath, {id: String(node.key), name: node.label}];

		// 如果找到目标节点，返回完整路径
		if (String(node.key) === targetId) {
			return newPath;
		}

		// 如果有子节点，递归查找
		if (node.children && node.children.length > 0) {
			const foundPath = findNodePath(node.children, targetId, newPath);
			if (foundPath) {
				return foundPath;
			}
		}
	}
	return null;
}

// 当前路径
const currentPath = computed(() => {
	// 根路径始终是路径的起点
	const rootPath = [{id: '0', name: t('bookmarkManager.rootDirectory') }];

	// 如果没有选中文件夹或选中的是根文件夹，只返回根路径
	if (!selectedFolder.value || selectedFolder.value === '0') {
		return rootPath;
	}

	// 查找选中文件夹的完整路径
	const fullPath = findNodePath(bookmarkTree.value, selectedFolder.value, []);

	// 如果找到完整路径，将根路径与子路径合并
	if (fullPath) {
		return rootPath.concat(fullPath);
	}

	// 如果找不到，返回根路径 + 当前选中文件夹
	return rootPath.concat([{id: selectedFolder.value, name: '...'}]);
});

// 处理面包屑点击
function handleBreadcrumbClick(id: string) {
	selectedFolder.value = id;
	selectedBookmarkId.value = '';
}

// 点击树节点
function handleSelect(keys: (string | number)[]) {
	// 更新选中的节点键引用
	selectedKeysRef.value = keys;

	// 重置选中的书签ID
	selectedBookmarkId.value = '';

	// 确保类型安全的赋值方式
	if (keys && Array.isArray(keys) && keys.length > 0) {
		const key = String(keys[0]);

		// 查找当前点击的节点
		function findNodeById(nodes: any[], id: string): any | null {
			for (const node of nodes) {
				if (String(node.key) === id) {
					return node;
				}
				if (node.children && node.children.length > 0) {
					const found = findNodeById(node.children, id);
					if (found) return found;
				}
			}
			return null;
		}

		const selectedNode = findNodeById(bookmarkTree.value, key);

		// 确保node存在
		if (selectedNode) {
			// 检查是否为具体书签节点
			if (selectedNode.isLeaf && selectedNode.bookmark) {
				// 如果是具体书签，设置selectedBookmarkId
				selectedBookmarkId.value = key;
				selectedFolder.value = ''; // 清空选中的文件夹
			}
			// 如果是文件夹，设置selectedFolder
			else if (selectedNode.isFolder || !selectedNode.isLeaf || !selectedNode.bookmark?.url) {
				selectedFolder.value = key;
				// 强制保持选中状态
				selectedKeysRef.value = [key];
			}
		}
	}
	// 不在这里清空selectedFolder，这样当取消选择时仍能保持文件夹筛选状态
	// 只有当选择了其他类型节点或明确取消选择时才清空
}

// 搜索时清空选中
function handleSearch() {
	if (searchQuery.value) {
		selectedFolder.value = ''
		selectedBookmarkId.value = '' // 同时清空选中的书签
	}
}



// 右键菜单相关状态
const isContextMenuOpen = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);
const currentBookmark = ref<(Bookmark & { isFolder?: boolean }) | null>(null);

// 树组件引用
const treeRef = ref<InstanceType<typeof NTree> | null>(null);

// 计算属性 - 所有文件夹（用于下拉选择）
const allFolders = computed(() => {
    const folders: { label: string; value: string }[] = [{ label: t('bookmarkManager.rootDirectory'), value: '0' }];
    const collectFolders = (nodes: any[]) => {
        for (const node of nodes) {
            if (!node.isLeaf) {
                folders.push({ label: node.label, value: String(node.key) });
                if (node.children) {
                    collectFolders(node.children);
                }
            }
        }
    };
    collectFolders(bookmarkTree.value);
    return folders;
});



// 编辑书签相关
const isEditDialogOpen = ref(false);
const isCreateMode = ref(false); // 是否为创建模式
const bookmarkType = ref<'bookmark' | 'folder'>('bookmark'); // 书签类型
const currentEditBookmark = ref({
	id: 0,
	title: '',
	url: '',
	folderId: '0' as string | number | undefined,
} as { id: number; title: string; url: string; folderId?: string | number | undefined; });


// 右键菜单样式
const contextMenuStyle = computed(() => {
  // 菜单默认宽度和高度估算
  const menuWidth = 160; // 对应w-40
  const menuHeight = 80; // 估算两个菜单项的高度

  // 获取屏幕尺寸
  const screenWidth = window.innerWidth;
  const screenHeight = window.innerHeight;

  // 计算菜单位置，确保不超出屏幕
  let left = contextMenuX.value;
  let top = contextMenuY.value;

  // 如果菜单右侧会超出屏幕，调整左侧位置
  if (left + menuWidth > screenWidth) {
    left = screenWidth - menuWidth;
  }

  // 如果菜单底部会超出屏幕，调整顶部位置
  if (top + menuHeight > screenHeight) {
    top = screenHeight - menuHeight;
  }

  // 确保菜单位置不会小于0
  left = Math.max(0, left);
  top = Math.max(0, top);

  return {
    top: `${top}px`,
    left: `${left}px`,
  };
});

// 打开右侧列表右键菜单
function openContextMenu(event: MouseEvent, bookmark: Bookmark & { isFolder?: boolean }) {
	event.preventDefault()
	event.stopPropagation()
	isContextMenuOpen.value = true
	contextMenuX.value = event.clientX
	contextMenuY.value = event.clientY
	currentBookmark.value = bookmark
}

// 双击进入文件夹
function enterFolder(folderId: string) {
	selectedFolder.value = folderId
	selectedBookmarkId.value = ''
	// 同步左侧树的选中状态
	selectedKeysRef.value = [folderId]
	// 刷新显示
	refreshBookmarks()
}

// 左侧树节点右键菜单
function handleTreeContextMenu({ node, event }: { node: any; event: MouseEvent }) {
	event.preventDefault()
	event.stopPropagation()

	// 打开菜单
	isContextMenuOpen.value = true
	contextMenuX.value = event.clientX
	contextMenuY.value = event.clientY

	// 查找当前节点的父文件夹ID
	let parentFolderId = '0'; // 默认根目录

	// 如果是从bookmark对象中可以直接获取folderId
	if (node.bookmark?.folderId) {
		parentFolderId = node.bookmark.folderId;
	} else {
		// 否则尝试通过遍历树结构找到父节点
		function findParentId(treeNodes: any[], targetId: string | number): string {
			for (const item of treeNodes) {
				if (item.children) {
					// 检查当前节点的子节点中是否有目标节点
					const childFound = item.children.some((child: any) => String(child.key) === String(targetId));
					if (childFound) {
						return String(item.key); // 返回找到的父节点ID，确保是字符串类型
					}
					// 递归搜索子节点
					const parentId = findParentId(item.children, targetId);
					if (parentId !== '0') {
						return parentId; // parentId已经是字符串类型
					}
				}
			}
			return '0'; // 确保返回字符串类型
		}
		parentFolderId = findParentId(bookmarkTree.value, String(node.key));
	}

	// 当前节点信息
	currentBookmark.value = {
		id: node.key,
		title: node.label,
		url: node.bookmark?.url || '',
		folderId: parentFolderId, // 使用找到的父文件夹ID
		isFolder: !node.isLeaf,
	}
}

// 渲染标签函数，使用SVG图标
// 自定义折叠图标渲染函数
const renderExpandIcon = ({ option }: { option: TreeOption }) => {
	// 现在我们已经通过设置isLeaf属性来控制折叠图标显示
	// 这里保持简单，让Tree组件根据isLeaf属性自动决定
	return undefined;
};

const renderTreeLabel = ({ option }: { option: any }) => {
  // 检查是否是文件夹节点
  const isFolder = option.isFolder || (!option.isLeaf && !option.bookmark?.url);

  // 尝试从不同的可能属性获取标题，增加健壮性
  const nodeTitle = option.title || option.label || option.name || '未命名';

  // 创建内容数组
  const content = [];

  // 如果是文件夹节点，添加SVG图标
  if (isFolder) {
    content.push(
      h('img', {
        src: FolderOpenIcon,
        class: 'w-5 h-5 inline-block mr-1',
        alt: 'folder icon'
      })
    );

    // 显示文件夹标题
    content.push(nodeTitle);
  } else {
    // 非文件夹节点只显示标题
    content.push(nodeTitle);
  }

  return h(
    'div',
    {
      class: 'px-1 py-0.5 rounded hover:bg-gray-100 cursor-default flex items-center',
      onContextmenu: (e: MouseEvent) => {
        handleTreeContextMenu({ node: option, event: e })
      },
    },
    content
  );
}

function handleGlobalClick(event: MouseEvent) {
	const path = event.composedPath() as HTMLElement[]
	const clickedInsideMenu = path.some(
		(el) => el.classList && el.classList.contains('context-menu')
	)
	if (!clickedInsideMenu) {
		closeContextMenu()
	}
}


// 关闭右键菜单
function closeContextMenu() {
	isContextMenuOpen.value = false;
}


// 打开书签URL
function openBookmark(bookmark: Bookmark) {
	if (bookmark.url && !bookmark.isFolder) {
		// 使用window.open在新标签页中打开书签URL
		window.open(bookmark.url, '_blank');
	}
}

// 处理编辑书签
function handleEditBookmark() {
  if (currentBookmark.value) {
    currentEditBookmark.value = {
      ...currentBookmark.value,
    };
    // 根据是否为文件夹设置书签类型
    bookmarkType.value = currentBookmark.value.isFolder ? 'folder' : 'bookmark';
    // 设置为修改模式
    isCreateMode.value = false;
    isEditDialogOpen.value = true;
  }
  isContextMenuOpen.value = false;
}

// 处理删除书签
function handleDeleteBookmark() {
	if (currentBookmark.value) {
		deleteBookmark(currentBookmark.value);
	}
	isContextMenuOpen.value = false;
}

// 创建新书签
function createNewBookmark() {
	// 重置表单
	currentEditBookmark.value = {
		id: 0,
		title: '',
		url: '',
		folderId: '0',
	};
	// 设置为创建模式
	isCreateMode.value = true;
	// 默认选择书签类型
	bookmarkType.value = 'bookmark';
	// 打开编辑对话框
	isEditDialogOpen.value = true;
}

// 关闭编辑对话框
function closeEditDialog() {
	isEditDialogOpen.value = false;
}

// 保存书签修改
async function saveBookmarkChanges() {
	try {
		// 数据验证
		if (!currentEditBookmark.value.title.trim()) {
			ms.error(t('bookmarkManager.titleRequired'));
			return;
		}

		// URL验证和处理 - 只有在编辑书签或创建书签时才需要URL
		// 修改模式
		// 根据是否为文件夹构建不同的更新数据
		const isFolderItem = bookmarkType.value === 'folder';

		// 如果不是文件夹且需要URL但未提供，显示错误
		if (!isFolderItem && !currentEditBookmark.value.url.trim()) {
			ms.error(t('bookmarkManager.urlRequired'));
			return;
		}

		// 只有书签才需要添加URL协议检查和补充
		if (!isFolderItem && currentEditBookmark.value.url.trim()) {
			let url = currentEditBookmark.value.url.trim();
			if (!url.startsWith('http://') && !url.startsWith('https://')) {
				url = 'https://' + url;
				currentEditBookmark.value.url = url;
			}
		}

		// 根据模式决定调用哪个接口
		if (isCreateMode.value) {
			// 创建新模式
			const createData = {
				title: currentEditBookmark.value.title,
				url: '',
				// 注意后端JSON标签是小写的parentUrl
				parentUrl: currentEditBookmark.value.folderId ? currentEditBookmark.value.folderId.toString() : '0',
				sort: 9999,
				lanUrl: '',
				icon: null,
				openMethod: 0,
				// IconJson在后端被标记为json:"-",不参与JSON序列化
			};

			// 根据类型添加相应的字段
			if (bookmarkType.value === 'folder') {
				// 文件夹：设置isFolder为1，URL设为标题
				Object.assign(createData, {
					isFolder: 1,
					url: currentEditBookmark.value.title
				});
			} else {
				// 书签：添加URL
				Object.assign(createData, {
					isFolder: 0,
					url: currentEditBookmark.value.url
				});
			}

			// 调用添加接口
			const createResponse = await addBookmark(createData);

			// 检查响应状态
				if (createResponse && createResponse.code === 0) {
					await refreshBookmarks();
					ms.success(t('bookmarkManager.createSuccess'));
					isEditDialogOpen.value = false;
					isCreateMode.value = false;
				} else {
					ms.error(`${t('bookmarkManager.createFailed')} ${createResponse?.msg || t('bookmarkManager.unknownError')}`);
				}
		} else {
			// 修改模式
			// 根据是否为文件夹构建不同的更新数据
			const isFolderItem = currentBookmark.value?.isFolder;
			const updateData = {
				id: Number(currentEditBookmark.value.id),
				title: currentEditBookmark.value.title,
				parentUrl: currentEditBookmark.value.folderId ? currentEditBookmark.value.folderId.toString() : '0',
				sort: 9999,
				lanUrl: '',
				icon: null,
				openMethod: 0,
				url: '' // 添加url属性以满足bookmarkInfo类型要求
			};
			
			// 文件夹特殊处理：将URL设置为标题或空
			if (isFolderItem) {
				updateData.url = currentEditBookmark.value.title;
			} else {
				// 书签正常处理URL
				updateData.url = currentEditBookmark.value.url;
			}
			
			const updateResponse = await updateBookmark(updateData);

				// 检查响应状态
				if (updateResponse && updateResponse.code === 0) {
					await refreshBookmarks();
					ms.success(t('bookmarkManager.updateSuccess'));
					isEditDialogOpen.value = false;
				} else {
					ms.error(`${t('bookmarkManager.updateFailed')} ${updateResponse?.msg || t('bookmarkManager.unknownError')}`);
				}
			}
	} catch (error) {
		console.error('保存书签失败', error);
		ms.error(`${t('bookmarkManager.updateFailed')} ${(error as Error).message || t('common.networkError')}`);
	}
}

// 删除书签或文件夹
async function deleteBookmark(bookmark: Bookmark) {
	// 根据是否为文件夹显示不同的确认消息
	const confirmMessage = bookmark.isFolder
		? t('bookmarkManager.deleteFolderConfirm').replace('name', "【"+bookmark.title+"】")
		 : t('bookmarkManager.deleteBookmarkConfirm').replace('name', "【"+bookmark.title+"】");

	// 直接使用从apiMessage导入的dialog对象
	dialog.warning({
		title: t('bookmarkManager.confirmDelete'),
		content: confirmMessage,
		positiveText: t('bookmarkManager.confirm'),
		negativeText: t('bookmarkManager.cancel'),
		onPositiveClick: async () => {
			 try {
				 const response = await deletes([Number(bookmark.id)]);
				 if (response.code === 0) {
					 // 清除所有可能的缓存
					 ss.remove(BOOKMARKS_CACHE_KEY);
					 ss.remove(BOOKMARKS_FULL_CACHE_KEY);
					 ss.remove('bookmark_tree_cache');
					 ss.remove('bookmark_list_cache');

					 // 重置选中状态
					 if (selectedBookmarkId.value === bookmark.id.toString()) {
						 selectedBookmarkId.value = '';
					 }

					 // 关键修改：清空现有的响应式数据
					 bookmarkTree.value = [];
					 // 使用类型断言来安全地访问和删除全局变量
					 if ((globalThis as any).__bookmarksFullData) {
						 delete (globalThis as any).__bookmarksFullData;
					 }
					 // 清空响应式的fullData变量
					 fullData.value = [];

					 // 强制刷新数据
					 await refreshBookmarks(true);

					 // 使用setTimeout确保DOM更新后显示成功消息
					 setTimeout(() => {
						 ms.success(t('common.success'));
					 }, 500); // 增加延迟时间
				 } else {
					 ms.error(`${t('common.failed')}: ${response.msg}`);
				 }
			 } catch (error) {
				 ms.error(`${t('common.failed')} ${(error as Error).message || t('bookmarkManager.unknownError')}`);
			 }
		 }
	});
}

// 触发导入书签
function triggerImportBookmarks() {
	fileInput.value?.click();
}

// 处理文件选择变化
function handleFileChange(event: Event) {
	const target = event.target as HTMLInputElement;
	const file = target.files?.[0];

	if (file) {
		uploadLoading.value = true;
		const reader = new FileReader();
		reader.onload = (e) => {
			if (e.target?.result) {
				jsonData.value = e.target.result as string;
				importCheck(file.name);
			} else {
				ms.error(`${t('common.failed')}: ${t('common.repeatLater')}`);
				uploadLoading.value = false;
			}
		};
		reader.onerror = () => {
			ms.error(`${t('common.failed')}: ${t('common.fileReadError')}`);
			uploadLoading.value = false;
		};
		reader.readAsText(file);
	} else {
		uploadLoading.value = false;
	}

	// 清空input的值，以便可以重复选择同一个文件
	target.value = '';
}

// 验证导入文件
function importCheck(fileName: string) {
	importWarning.value = [];

	try {
		if (fileName.endsWith('.html')) {
			// 直接将HTML内容传递给后端解析
			importBookmarksToServerWithHTML(jsonData.value!);
		} else {
			ms.error(t('bookmarkManager.onlySupportHtml'));
		}
	} catch (error) {
		ms.error(`${t('common.failed')}: ${(error as Error).message || t('common.unknownError')}`);
	} finally {
		uploadLoading.value = false;
	}
}

// 导入HTML书签到服务器
async function importBookmarksToServerWithHTML(htmlContent: string) {
	uploadLoading.value = true;
	try {
		// 直接将HTML内容传递给后端
		const response = await addMultipleBookmarks({ htmlContent } as any);
		if (response.code === 0) {
			ms.success(t('bookmarkManager.importSuccess').replace('count', (response.data as any).count));
			ss.remove(BOOKMARKS_CACHE_KEY)
			// 刷新书签列表
			await refreshBookmarks();
		} else {
			ms.error(`${t('common.failed')}: ${response.msg}`);
		}
	} catch (error) {
		ms.error(`${t('common.failed')}: ${(error as Error).message || t('common.unknownError')}`);
	} finally {
		uploadLoading.value = false;
	}
}


// 过滤函数：从完整树结构中只保留文件夹
function filterFoldersOnly(nodes: TreeOption[]): TreeOption[] {
	return nodes.map(node => {
		// 克隆节点时，确保保留 disabledExpand（...node 会自动浅拷贝）
		const folderNode: TreeOption = {
			...node,
			bookmark: undefined,
			children: []
		};

		if (Array.isArray(node.children) && node.children.length > 0) {
			const folderChildren = node.children.filter(child => child.isFolder);
			if (folderChildren.length > 0) {
				folderNode.children = filterFoldersOnly(folderChildren);
				// 子节点不为空 → 启用折叠，设置isLeaf为false
				folderNode.isLeaf = false;
				folderNode.disabledExpand = false;
			} else {
				folderNode.children = [];
				// 子节点为空 → 禁用折叠，设置isLeaf为true
				folderNode.isLeaf = true;
				folderNode.disabledExpand = true;
			}
		} else {
			// 没有子节点 → 设置isLeaf为true
			folderNode.isLeaf = true;
			folderNode.disabledExpand = true;
		}

		return folderNode;
	}).filter(node => node.isFolder);
}

// 刷新书签列表
async function refreshBookmarks(forceRefresh = false) {
	try {
		// 强制刷新模式下，直接跳过缓存检查
		if (!forceRefresh) {
			// 尝试从缓存读取完整数据和文件夹树结构
			const cachedFullData = ss.get(BOOKMARKS_FULL_CACHE_KEY);
			const cachedFolderTree = ss.get(BOOKMARKS_CACHE_KEY);

			if (cachedFullData && cachedFolderTree) {
				// 直接使用缓存的文件夹树结构
				bookmarkTree.value = cachedFolderTree;

				// 设置默认展开第一级节点
				defaultExpandedKeys.value = bookmarkTree.value.map(node => node.key);

				// 同时更新响应式的fullData变量
				fullData.value = cachedFullData;

				// 保留全局变量的更新，保证向后兼容
				if (globalThis) {
					Object.defineProperty(globalThis, '__bookmarksFullData', { value: cachedFullData, configurable: true });
				}
				return;
			}
		}

		// 缓存中没有数据或强制刷新，请求接口获取数据
		const response = await getBookmarksList();
		if (response.code === 0) {
			// 检查数据结构，如果已经是树形结构则直接使用
			const data: any = response.data || [];

			// 检查是否已经是树形结构（直接包含children字段）
			let fullTreeData = [];
			if (Array.isArray(data) && data.length > 0 && 'children' in data[0]) {
				// 已经是树形结构，转换为前端需要的格式
				fullTreeData = convertServerTreeToFrontendTree(data);
			} else if (data.list && Array.isArray(data.list)) {
				// 后端返回的是带list字段的结构
				const serverBookmarks = data.list;
				if (serverBookmarks.length > 0 && 'children' in serverBookmarks[0]) {
					// list字段中已经是树形结构
					fullTreeData = convertServerTreeToFrontendTree(serverBookmarks);
				} else {
					// 构建树形结构
					fullTreeData = buildBookmarkTree(serverBookmarks);
				}
			} else {
				// 作为列表数据构建树形结构
				fullTreeData = buildBookmarkTree(Array.isArray(data) ? data : []);
			}

			// 过滤出只包含文件夹的树结构
			const folderTreeData = filterFoldersOnly(fullTreeData);

			// 更新响应式数据
			bookmarkTree.value = folderTreeData;
			fullData.value = fullTreeData;

			// 设置默认展开第一级节点
			defaultExpandedKeys.value = bookmarkTree.value.map(node => node.key);

			// 分别缓存完整数据和文件夹树结构
			ss.set(BOOKMARKS_FULL_CACHE_KEY, fullTreeData);
			ss.set(BOOKMARKS_CACHE_KEY, folderTreeData);

			// 保留全局变量的更新，保证向后兼容
			if (globalThis) {
				Object.defineProperty(globalThis, '__bookmarksFullData', { value: fullTreeData, configurable: true });
			}
		}
	} catch (error) {
		console.error('刷新书签列表失败:', error);
		// 发生错误时清空响应式数据，避免显示旧数据
		bookmarkTree.value = [];
		fullData.value = [];
	}
}

// 将服务器返回的树形结构转换为前端组件需要的格式
function convertServerTreeToFrontendTree(serverTree: any[]): TreeOption[] {
	// 为每个节点创建一个映射来存储父ID信息
	const folderIdMap = new Map<string, string | null>();

	const result = serverTree.map(node => {
		let bookmarkObj = undefined;
		if (node.isFolder !== 1 && node.url) {
			const folderId = node.ParentUrl !== undefined ? String(node.ParentUrl) : null;
			// 存储在单独的映射中而不是节点对象上
			folderIdMap.set(String(node.id), folderId);
			bookmarkObj = { id: node.id, title: node.title, url: node.url };
		}

		// 计算是否有子节点
		const hasChildren = Array.isArray(node.children) && node.children.length > 0;

		const frontendNode: TreeOption = {
			key: String(node.id),
			label: node.title || '未命名',
			// 关键修改：对于文件夹节点，如果没有子节点则设置isLeaf为true
			isLeaf: node.isFolder !== 1 || !hasChildren,
			isFolder: node.isFolder === 1,
			bookmark: bookmarkObj,
			children: hasChildren ? convertServerTreeToFrontendTree(node.children) : [],
			rawNode: node,
			disabledExpand: !hasChildren
		};

		return frontendNode;
	});

	// 在处理完所有节点后，处理父子关系
	// 注意：这里我们只需要返回转换后的节点，父子关系会在外部处理
	return result;
}

function buildBookmarkTree(bookmarks: any[]): TreeOption[] {
	const folderMap = new Map<string, TreeOption>();
	const allNodes: TreeOption[] = [];
	// 使用单独的映射来存储folderId信息
	const folderIdMap = new Map<string, string>();

	// 第一步：创建所有节点
	for (const bookmark of bookmarks) {
		const isFolder = bookmark.isFolder === 1;
		const nodeKey = String(bookmark.id || bookmark.Key || '0');
		const node: TreeOption = {
			key: nodeKey,
			label: bookmark.title || '未命名',
			isLeaf: !isFolder,
			isFolder: isFolder,
			rawNode: bookmark,
			children: [],
			disabledExpand: true // 初始设为true，后续根据实际情况调整
		};

		if (isFolder) {
			node.bookmark = undefined;
			folderMap.set(node.key, node);
		} else {
			// 创建不包含folderId的bookmark对象
			const { folderId, ...bookmarkWithoutFolderId } = bookmark;
			node.bookmark = bookmarkWithoutFolderId;
			// 将folderId存储在单独的映射中
			if (folderId) {
				folderIdMap.set(nodeKey, String(folderId));
			}
		}

		allNodes.push(node);
	}

	// 第二步：构建树形结构
	const rootNodes: TreeOption[] = [];

	// 构建父子关系
	for (const node of allNodes) {
		if (node.isFolder) continue;

		// 获取父文件夹ID - 从单独的映射中获取
		const parentId = folderIdMap.get(node.key) || '0';
		const parentFolder = folderMap.get(String(parentId));

		if (parentFolder) {
			// 添加到父文件夹的子节点列表
			parentFolder.children.push(node);
			// 更新父文件夹的disabledExpand属性
			parentFolder.disabledExpand = false;
		} else {
			// 如果没有父文件夹，添加到根节点
			rootNodes.push(node);
		}
	}

	// 添加所有根文件夹到根节点列表
	for (const node of folderMap.values()) {
		// 检查是否是根文件夹（没有被添加为其他节点的子节点）
		const isRootFolder = !Array.from(folderMap.values()).some(parent =>
			parent.children.some(child => child.key === node.key)
		);

		if (isRootFolder) {
			// 更新根文件夹的disabledExpand属性
			node.disabledExpand = node.children.length === 0;
			rootNodes.push(node);
		}
	}

	// 第三步：更新文件夹节点的isLeaf和disabledExpand属性
	for (const node of allNodes) {
		if (node.isFolder) {
			// 关键修改：空文件夹设置isLeaf为true，但保持isFolder为true
			// 这样Tree组件不会显示折叠图标，但renderTreeLabel仍会将其识别为文件夹
			const hasChildren = Array.isArray(node.children) && node.children.length > 0;
			node.isLeaf = !hasChildren;
			node.disabledExpand = !hasChildren;
		}
	}

	return rootNodes;
}

const handleResize = () => {
	isMobile.value = window.innerWidth < 768
	// 移动端默认隐藏左栏
	showLeftPanel.value = isMobile.value ? false : true
}
// 组件挂载时加载书签
onMounted(async () => {
	ss.remove(BOOKMARKS_CACHE_KEY)
	ss.remove(BOOKMARKS_FULL_CACHE_KEY)
	await refreshBookmarks();
	// 添加全局事件监听器
	document.addEventListener('mousemove', handleMouseMove);
	document.addEventListener('mouseup', stopResize);
	document.addEventListener('click', handleGlobalClick);

	handleResize()
	window.addEventListener('resize', handleResize)
});

onUnmounted(() => {
	document.removeEventListener('mousemove', handleMouseMove);
	document.removeEventListener('mouseup', stopResize);
	document.removeEventListener('click', handleGlobalClick);

	window.removeEventListener('resize', handleResize)
	// 清理全局存储的完整数据，避免内存泄漏
	if ((globalThis as any).__bookmarksFullData) {
		delete (globalThis as any).__bookmarksFullData;
	}
});



</script>

<style>
.context-menu {
position: fixed !important;
z-index: 99999 !important;
background-color: white;
}


/* 阴影和圆角让左栏浮动更美观 */
.fixed.bg-white {
	border-right: 1px solid #e5e7eb; /* 柔和边框 */
	border-radius: 0 0.75rem 0.75rem 0; /* 左边圆角 */
	box-shadow: 0 4px 12px rgba(0,0,0,0.1);
	transition: width 0.3s ease-in-out, transform 0.3s ease-in-out;
}
</style>