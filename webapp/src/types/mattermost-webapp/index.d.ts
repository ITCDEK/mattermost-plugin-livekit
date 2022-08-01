export interface PluginRegistry {
    registerPostTypeComponent(typeName: string, component: React.ElementType)
    registerReducer(reducer: Reducer)
    registerChannelHeaderButtonAction(component: React.Element, fn: (channel: Channel) => void, dropdownText: string, tooltipText: string)
    registerSlashCommandWillBePostedHook(hook: (message: string, args: CommandArgs) => any)
    //
    unregisterPostTypeComponent(componentID: string)

    // Add more if needed from https://developers.mattermost.com/extend/plugins/webapp/reference
}
