local Device =  require("device")
local Dispatcher = require("dispatcher")
local InfoMessage = require("ui/widget/infomessage")  -- luacheck:ignore
local UIManager = require("ui/uimanager")
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local ffiutil = require("ffi/util")
local logger = require("logger")
local util = require("util")
local _ = require("gettext")
local T = ffiutil.template

-- Plugin uses custom written golang-based server

if not util.fileExists("plugins/m4k.koplugin/m4k_receiver") then
    return { disabled = true, }
end

local M4KReceiver = WidgetContainer:extend{
    name = "m4k",
    is_doc_only = false,
}

function M4KReceiver:init()
    self.port = "49494"
    self.log_file_path = "/mnt/us/koreader/m4k_receiver_log.txt"
    self.ui.menu:registerToMainMenu(self)
    self:onDispatcherRegisterActions()
end

function M4KReceiver:start()
    local cmd = string.format("./plugins/m4k.koplugin/m4k_receiver -port %s -pidfile %s >%s 2>&1 &",
        self.port,
        "/tmp/m4k_receiver_koreader.pid",
        self.log_file_path)

    -- Make a hole in the Kindle's firewall
    if Device:isKindle() then
        os.execute(string.format("%s %s %s",
            "iptables -A INPUT -p tcp --dport", self.port,
            "-m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT"))
        os.execute(string.format("%s %s %s",
            "iptables -A OUTPUT -p tcp --sport", self.port,
            "-m conntrack --ctstate ESTABLISHED -j ACCEPT"))
    end

    logger.dbg("[Network] Launching m4k receiver : ", cmd)
    if os.execute(cmd) == 0 then
        local info = InfoMessage:new{
                timeout = 10,
                text = T(_("m4k receiver started.\n\nport: %1\n%2"),
                    self.port,
                    Device.retrieveNetworkInfo and Device:retrieveNetworkInfo() or _("Could not retrieve network info.")),
        }
        UIManager:show(info)
    else
        local info = InfoMessage:new{
                icon = "notice-warning",
                text = _("Failed to start m4k receiver."),
        }
        UIManager:show(info)
    end
end

function M4KReceiver:isRunning()
    return util.pathExists("/tmp/m4k_receiver_koreader.pid")
end

function M4KReceiver:stop()
    os.execute("cat /tmp/m4k_receiver_koreader.pid | xargs kill")
    UIManager:show(InfoMessage:new {
        text = T(_("m4k receiver stopped.")),
        timeout = 2,
    })

    if self:isRunning() then
        os.remove("/tmp/m4k_receiver_koreader.pid")
    end

    -- Plug the hole in the Kindle's firewall
    if Device:isKindle() then
        os.execute(string.format("%s %s %s",
            "iptables -D INPUT -p tcp --dport", self.port,
            "-m conntrack --ctstate NEW,ESTABLISHED -j ACCEPT"))
        os.execute(string.format("%s %s %s",
            "iptables -D OUTPUT -p tcp --sport", self.port,
            "-m conntrack --ctstate ESTABLISHED -j ACCEPT"))
    end
end

function M4KReceiver:onToggleM4KReceiver()
    if self:isRunning() then
        self:stop()
    else
        self:start()
    end
end

function M4KReceiver:addToMainMenu(menu_items)
    menu_items.m4k = {
        text = _("m4k"),
        sub_item_table = {
            {
                text = _("Receiver"),
                keep_menu_open = true,
                checked_func = function() return self:isRunning() end,
                callback = function(touchmenu_instance)
                    self:onToggleM4KReceiver()
                    -- sleeping might not be needed, but it gives the feeling
                    -- something has been done and feedback is accurate
                    ffiutil.sleep(1)
                    touchmenu_instance:updateItems()
                end,
            },
       }
    }
end

function M4KReceiver:onDispatcherRegisterActions()
    Dispatcher:registerAction("toggle_m4k_receiver", {
        category = "none",
        event = "ToggleM4KReceiver",
        title = _("Toggle m4k receiver"),
        general=true
    })
end

return M4KReceiver
