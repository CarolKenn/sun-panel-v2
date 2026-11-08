// 前端 localStorage 缓存 key 统一在此维护。
//
// 历史问题：缓存清理分散在各组件里手写字符串，出现了
// 'GROUP_LIST_CACHE' / 'GROUP_LIST_CACHE_KEY' 这类与真实 key 'groupListCache'
// 不一致的写法，清理动作静默失效。首页又是「缓存优先，命中就直接 return」，
// 结果就是分组增删改后回首页看不到变化。新增缓存时请复用这里的常量。

// 面板分组列表
export const CACHE_KEY_GROUP_LIST = 'groupListCache'
// 书签树
export const CACHE_KEY_BOOKMARKS_TREE = 'bookmarksTreeCache'
// 分组下的图标列表（真实 key = 前缀 + 分组 id）
export const CACHE_KEY_ITEM_ICON_LIST_PREFIX = 'itemIconList_'
// 当前登录用户信息
export const CACHE_KEY_USER_AUTH_INFO = 'USER_AUTH_INFO_CACHE'
// 用户面板配置（样式等）
export const CACHE_KEY_USER_CONFIG = 'USER_CONFIG_CACHE'
