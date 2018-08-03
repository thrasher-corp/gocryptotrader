package slack

// WebsocketResponse holds websocket response data
type WebsocketResponse struct {
	Type    string `json:"type"`
	ReplyTo int    `json:"reply_to"`
	Error   struct {
		Msg  string `json:"msg"`
		Code int    `json:"code"`
	} `json:"error"`
}

// SendMessage holds details for message information
type SendMessage struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// Message is a response type handling message data
type Message struct {
	Channel    string  `json:"channel"`
	User       string  `json:"user"`
	Text       string  `json:"text"`
	SourceTeam string  `json:"source_team"`
	Timestamp  float64 `json:"ts,string"`
	Team       string  `json:"team"`
}

// PresenceChange holds user presence data
type PresenceChange struct {
	Presence string `json:"presence"`
	User     string `json:"user"`
}

// Response is a generalised response type
type Response struct {
	Bots []struct {
		AppID   string `json:"app_id"`
		Deleted bool   `json:"deleted"`
		Icons   struct {
			Image36 string `json:"image_36"`
			Image48 string `json:"image_48"`
			Image72 string `json:"image_72"`
		} `json:"icons"`
		ID      string `json:"id"`
		Name    string `json:"name"`
		Updated int    `json:"updated"`
	} `json:"bots"`
	CacheTs                 int    `json:"cache_ts"`
	CacheTsVersion          string `json:"cache_ts_version"`
	CacheVersion            string `json:"cache_version"`
	CanManageSharedChannels bool   `json:"can_manage_shared_channels"`
	Channels                []struct {
		Created        int      `json:"created"`
		Creator        string   `json:"creator"`
		HasPins        bool     `json:"has_pins"`
		ID             string   `json:"id"`
		IsArchived     bool     `json:"is_archived"`
		IsChannel      bool     `json:"is_channel"`
		IsGeneral      bool     `json:"is_general"`
		IsMember       bool     `json:"is_member"`
		IsOrgShared    bool     `json:"is_org_shared"`
		IsShared       bool     `json:"is_shared"`
		Name           string   `json:"name"`
		NameNormalized string   `json:"name_normalized"`
		PreviousNames  []string `json:"previous_names"`
	} `json:"channels"`
	Dnd struct {
		DndEnabled     bool `json:"dnd_enabled"`
		NextDndEndTs   int  `json:"next_dnd_end_ts"`
		NextDndStartTs int  `json:"next_dnd_start_ts"`
		SnoozeEnabled  bool `json:"snooze_enabled"`
	} `json:"dnd"`
	Groups []struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		IsGroup        bool     `json:"is_group"`
		Created        int64    `json:"created"`
		Creator        string   `json:"creator"`
		IsArchived     bool     `json:"is_archived"`
		NameNormalised string   `json:"name_normalised"`
		IsMPIM         bool     `json:"is_mpim"`
		HasPins        bool     `json:"has_pins"`
		IsOpen         bool     `json:"is_open"`
		LastRead       string   `json:"last_read"`
		Members        []string `json:"members"`
		Topic          struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int64  `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int64  `json:"last_set"`
		} `json:"purpose"`
	} `json:"groups"`
	Ims []struct {
		Created     int    `json:"created"`
		HasPins     bool   `json:"has_pins"`
		ID          string `json:"id"`
		IsIm        bool   `json:"is_im"`
		IsOpen      bool   `json:"is_open"`
		IsOrgShared bool   `json:"is_org_shared"`
		LastRead    string `json:"last_read"`
		User        string `json:"user"`
	} `json:"ims"`
	LatestEventTs    string        `json:"latest_event_ts"`
	Ok               bool          `json:"ok"`
	Error            string        `json:"error"`
	ReadOnlyChannels []interface{} `json:"read_only_channels"`
	Self             struct {
		Created        int    `json:"created"`
		ID             string `json:"id"`
		ManualPresence string `json:"manual_presence"`
		Name           string `json:"name"`
		Prefs          struct {
			A11yAnimations                     bool        `json:"a11y_animations"`
			A11yFontSize                       string      `json:"a11y_font_size"`
			AllChannelsLoud                    bool        `json:"all_channels_loud"`
			AllNotificationsPrefs              string      `json:"all_notifications_prefs"`
			AllUnreadsSortOrder                string      `json:"all_unreads_sort_order"`
			AllowCallsToSetCurrentStatus       bool        `json:"allow_calls_to_set_current_status"`
			AnalyticsUpsellCoachmarkSeen       bool        `json:"analytics_upsell_coachmark_seen"`
			ArrowHistory                       bool        `json:"arrow_history"`
			AtChannelSuppressedChannels        string      `json:"at_channel_suppressed_channels"`
			BoxEnabled                         bool        `json:"box_enabled"`
			ChannelSort                        string      `json:"channel_sort"`
			ClientLogsPri                      string      `json:"client_logs_pri"`
			ColorNamesInList                   bool        `json:"color_names_in_list"`
			ConfirmClearAllUnreads             bool        `json:"confirm_clear_all_unreads"`
			ConfirmShCallStart                 bool        `json:"confirm_sh_call_start"`
			ConfirmUserMarkedAway              bool        `json:"confirm_user_marked_away"`
			ConvertEmoticons                   bool        `json:"convert_emoticons"`
			DisplayDisplayNames                bool        `json:"display_display_names"`
			DisplayRealNamesOverride           int         `json:"display_real_names_override"`
			DndEnabled                         bool        `json:"dnd_enabled"`
			DndEndHour                         string      `json:"dnd_end_hour"`
			DndStartHour                       string      `json:"dnd_start_hour"`
			DropboxEnabled                     bool        `json:"dropbox_enabled"`
			EmailAlerts                        string      `json:"email_alerts"`
			EmailAlertsSleepUntil              int         `json:"email_alerts_sleep_until"`
			EmailMisc                          bool        `json:"email_misc"`
			EmailWeekly                        bool        `json:"email_weekly"`
			EmojiAutocompleteBig               bool        `json:"emoji_autocomplete_big"`
			EmojiMode                          string      `json:"emoji_mode"`
			EmojiUse                           string      `json:"emoji_use"`
			EnableReactEmojiPicker             bool        `json:"enable_react_emoji_picker"`
			EnableUnreadView                   bool        `json:"enable_unread_view"`
			EnhancedDebugging                  bool        `json:"enhanced_debugging"`
			EnterIsSpecialInTbt                bool        `json:"enter_is_special_in_tbt"`
			EnterpriseMdmCustomMsg             string      `json:"enterprise_mdm_custom_msg"`
			EnterpriseMigrationSeen            bool        `json:"enterprise_migration_seen"`
			ExpandInlineImgs                   bool        `json:"expand_inline_imgs"`
			ExpandInternalInlineImgs           bool        `json:"expand_internal_inline_imgs"`
			ExpandNonMediaAttachments          bool        `json:"expand_non_media_attachments"`
			ExpandSnippets                     bool        `json:"expand_snippets"`
			FKeySearch                         bool        `json:"f_key_search"`
			FlannelServerPool                  string      `json:"flannel_server_pool"`
			FrecencyEntJumper                  string      `json:"frecency_ent_jumper"`
			FrecencyJumper                     string      `json:"frecency_jumper"`
			FullTextExtracts                   bool        `json:"full_text_extracts"`
			FullerTimestamps                   bool        `json:"fuller_timestamps"`
			GdriveAuthed                       bool        `json:"gdrive_authed"`
			GdriveEnabled                      bool        `json:"gdrive_enabled"`
			GraphicEmoticons                   bool        `json:"graphic_emoticons"`
			GrowlsEnabled                      bool        `json:"growls_enabled"`
			GrowthMsgLimitApproachingCtaCount  int         `json:"growth_msg_limit_approaching_cta_count"`
			GrowthMsgLimitApproachingCtaTs     int         `json:"growth_msg_limit_approaching_cta_ts"`
			GrowthMsgLimitLongReachedCtaCount  int         `json:"growth_msg_limit_long_reached_cta_count"`
			GrowthMsgLimitLongReachedCtaLastTs int         `json:"growth_msg_limit_long_reached_cta_last_ts"`
			GrowthMsgLimitReachedCtaCount      int         `json:"growth_msg_limit_reached_cta_count"`
			GrowthMsgLimitReachedCtaLastTs     int         `json:"growth_msg_limit_reached_cta_last_ts"`
			HasCreatedChannel                  bool        `json:"has_created_channel"`
			HasInvited                         bool        `json:"has_invited"`
			HasSearched                        bool        `json:"has_searched"`
			HasUploaded                        bool        `json:"has_uploaded"`
			HideHexSwatch                      bool        `json:"hide_hex_swatch"`
			HideUserGroupInfoPane              bool        `json:"hide_user_group_info_pane"`
			HighlightWords                     string      `json:"highlight_words"`
			IntroToAppsMessageSeen             bool        `json:"intro_to_apps_message_seen"`
			Jumbomoji                          bool        `json:"jumbomoji"`
			KKeyOmnibox                        bool        `json:"k_key_omnibox"`
			KKeyOmniboxAutoHideCount           int         `json:"k_key_omnibox_auto_hide_count"`
			LastSeenAtChannelWarning           int         `json:"last_seen_at_channel_warning"`
			LastSnippetType                    string      `json:"last_snippet_type"`
			LastTosAcknowledged                interface{} `json:"last_tos_acknowledged"`
			LoadLato2                          bool        `json:"load_lato_2"`
			Locale                             string      `json:"locale"`
			LoudChannels                       string      `json:"loud_channels"`
			LoudChannelsSet                    string      `json:"loud_channels_set"`
			LsDisabled                         bool        `json:"ls_disabled"`
			MacSsbBounce                       string      `json:"mac_ssb_bounce"`
			MacSsbBullet                       bool        `json:"mac_ssb_bullet"`
			MarkMsgsReadImmediately            bool        `json:"mark_msgs_read_immediately"`
			MeasureCSSUsage                    bool        `json:"measure_css_usage"`
			MentionsExcludeAtChannels          bool        `json:"mentions_exclude_at_channels"`
			MentionsExcludeAtUserGroups        bool        `json:"mentions_exclude_at_user_groups"`
			MessagesTheme                      string      `json:"messages_theme"`
			MsgPreview                         bool        `json:"msg_preview"`
			MsgPreviewPersistent               bool        `json:"msg_preview_persistent"`
			MuteSounds                         bool        `json:"mute_sounds"`
			MutedChannels                      string      `json:"muted_channels"`
			NeverChannels                      string      `json:"never_channels"`
			NewMsgSnd                          string      `json:"new_msg_snd"`
			NewxpSeenLastMessage               int         `json:"newxp_seen_last_message"`
			NoCreatedOverlays                  bool        `json:"no_created_overlays"`
			NoInvitesWidgetInSidebar           bool        `json:"no_invites_widget_in_sidebar"`
			NoJoinedOverlays                   bool        `json:"no_joined_overlays"`
			NoMacelectronBanner                bool        `json:"no_macelectron_banner"`
			NoMacssb1Banner                    bool        `json:"no_macssb1_banner"`
			NoMacssb2Banner                    bool        `json:"no_macssb2_banner"`
			NoOmniboxInChannels                bool        `json:"no_omnibox_in_channels"`
			NoTextInNotifications              bool        `json:"no_text_in_notifications"`
			NoWinssb1Banner                    bool        `json:"no_winssb1_banner"`
			ObeyInlineImgLimit                 bool        `json:"obey_inline_img_limit"`
			OnboardingCancelled                bool        `json:"onboarding_cancelled"`
			OnboardingSlackbotConversationStep int         `json:"onboarding_slackbot_conversation_step"`
			OverloadedMessageEnabled           bool        `json:"overloaded_message_enabled"`
			PagekeysHandled                    bool        `json:"pagekeys_handled"`
			PostsFormattingGuide               bool        `json:"posts_formatting_guide"`
			PreferredSkinTone                  string      `json:"preferred_skin_tone"`
			PrevNextBtn                        bool        `json:"prev_next_btn"`
			PrivacyPolicySeen                  bool        `json:"privacy_policy_seen"`
			PromptedForEmailDisabling          bool        `json:"prompted_for_email_disabling"`
			PushAtChannelSuppressedChannels    string      `json:"push_at_channel_suppressed_channels"`
			PushDmAlert                        bool        `json:"push_dm_alert"`
			PushEverything                     bool        `json:"push_everything"`
			PushIdleWait                       int         `json:"push_idle_wait"`
			PushLoudChannels                   string      `json:"push_loud_channels"`
			PushLoudChannelsSet                string      `json:"push_loud_channels_set"`
			PushMentionAlert                   bool        `json:"push_mention_alert"`
			PushMentionChannels                string      `json:"push_mention_channels"`
			PushShowPreview                    bool        `json:"push_show_preview"`
			PushSound                          string      `json:"push_sound"`
			QuestsEnabled                      bool        `json:"quests_enabled"`
			RequireAt                          bool        `json:"require_at"`
			SearchExcludeBots                  bool        `json:"search_exclude_bots"`
			SearchExcludeChannels              string      `json:"search_exclude_channels"`
			SearchOnlyCurrentTeam              bool        `json:"search_only_current_team"`
			SearchOnlyMyChannels               bool        `json:"search_only_my_channels"`
			SearchSort                         string      `json:"search_sort"`
			SeenAppSpaceCoachmark              bool        `json:"seen_app_space_coachmark"`
			SeenAppSpaceTutorial               bool        `json:"seen_app_space_tutorial"`
			SeenCallsSsMainCoachmark           bool        `json:"seen_calls_ss_main_coachmark"`
			SeenCallsSsWindowCoachmark         bool        `json:"seen_calls_ss_window_coachmark"`
			SeenCallsVideoBetaCoachmark        bool        `json:"seen_calls_video_beta_coachmark"`
			SeenCallsVideoGaCoachmark          bool        `json:"seen_calls_video_ga_coachmark"`
			SeenCustomStatusBadge              bool        `json:"seen_custom_status_badge"`
			SeenCustomStatusCallout            bool        `json:"seen_custom_status_callout"`
			SeenDomainInviteReminder           bool        `json:"seen_domain_invite_reminder"`
			SeenGdriveCoachmark                bool        `json:"seen_gdrive_coachmark"`
			SeenGuestAdminSlackbotAnnouncement bool        `json:"seen_guest_admin_slackbot_announcement"`
			SeenHighlightsArrowsCoachmark      bool        `json:"seen_highlights_arrows_coachmark"`
			SeenHighlightsCoachmark            bool        `json:"seen_highlights_coachmark"`
			SeenHighlightsWarmWelcome          bool        `json:"seen_highlights_warm_welcome"`
			SeenIntlChannelNamesCoachmark      bool        `json:"seen_intl_channel_names_coachmark"`
			SeenMemberInviteReminder           bool        `json:"seen_member_invite_reminder"`
			SeenOnboardingChannels             bool        `json:"seen_onboarding_channels"`
			SeenOnboardingDirectMessages       bool        `json:"seen_onboarding_direct_messages"`
			SeenOnboardingInvites              bool        `json:"seen_onboarding_invites"`
			SeenOnboardingPrivateGroups        bool        `json:"seen_onboarding_private_groups"`
			SeenOnboardingRecentMentions       bool        `json:"seen_onboarding_recent_mentions"`
			SeenOnboardingSearch               bool        `json:"seen_onboarding_search"`
			SeenOnboardingSlackbotConversation bool        `json:"seen_onboarding_slackbot_conversation"`
			SeenOnboardingStarredItems         bool        `json:"seen_onboarding_starred_items"`
			SeenOnboardingStart                bool        `json:"seen_onboarding_start"`
			SeenRepliesCoachmark               bool        `json:"seen_replies_coachmark"`
			SeenSingleEmojiMsg                 bool        `json:"seen_single_emoji_msg"`
			SeenSsbPrompt                      bool        `json:"seen_ssb_prompt"`
			SeenThreadsNotificationBanner      bool        `json:"seen_threads_notification_banner"`
			SeenUnreadViewCoachmark            bool        `json:"seen_unread_view_coachmark"`
			SeenWelcome2                       bool        `json:"seen_welcome_2"`
			SeparatePrivateChannels            bool        `json:"separate_private_channels"`
			SeparateSharedChannels             bool        `json:"separate_shared_channels"`
			ShowAllSkinTones                   bool        `json:"show_all_skin_tones"`
			ShowJumperScores                   bool        `json:"show_jumper_scores"`
			ShowMemoryInstrument               bool        `json:"show_memory_instrument"`
			ShowTyping                         bool        `json:"show_typing"`
			SidebarBehavior                    string      `json:"sidebar_behavior"`
			SidebarTheme                       string      `json:"sidebar_theme"`
			SidebarThemeCustomValues           string      `json:"sidebar_theme_custom_values"`
			SnippetEditorWrapLongLines         bool        `json:"snippet_editor_wrap_long_lines"`
			SpacesNewXpBannerDismissed         bool        `json:"spaces_new_xp_banner_dismissed"`
			SsEmojis                           bool        `json:"ss_emojis"`
			SsbSpaceWindow                     string      `json:"ssb_space_window"`
			StartScrollAtOldest                bool        `json:"start_scroll_at_oldest"`
			TabUIReturnSelects                 bool        `json:"tab_ui_return_selects"`
			ThreadsEverything                  bool        `json:"threads_everything"`
			Time24                             bool        `json:"time24"`
			TwoFactorAuthEnabled               bool        `json:"two_factor_auth_enabled"`
			TwoFactorBackupType                interface{} `json:"two_factor_backup_type"`
			TwoFactorType                      interface{} `json:"two_factor_type"`
			Tz                                 interface{} `json:"tz"`
			UseReactSidebar                    bool        `json:"use_react_sidebar"`
			UserColors                         string      `json:"user_colors"`
			WebappSpellcheck                   bool        `json:"webapp_spellcheck"`
			WelcomeMessageHidden               bool        `json:"welcome_message_hidden"`
			WhatsNewRead                       int         `json:"whats_new_read"`
			WinssbRunFromTray                  bool        `json:"winssb_run_from_tray"`
			WinssbWindowFlashBehavior          string      `json:"winssb_window_flash_behavior"`
		} `json:"prefs"`
	} `json:"self"`
	Subteams struct {
		All  []interface{} `json:"all"`
		Self []interface{} `json:"self"`
	} `json:"subteams"`
	Team struct {
		ApproachingMsgLimit bool   `json:"approaching_msg_limit"`
		AvatarBaseURL       string `json:"avatar_base_url"`
		Domain              string `json:"domain"`
		EmailDomain         string `json:"email_domain"`
		Icon                struct {
			Image102      string `json:"image_102"`
			Image132      string `json:"image_132"`
			Image230      string `json:"image_230"`
			Image34       string `json:"image_34"`
			Image44       string `json:"image_44"`
			Image68       string `json:"image_68"`
			Image88       string `json:"image_88"`
			ImageOriginal string `json:"image_original"`
		} `json:"icon"`
		ID                    string `json:"id"`
		MessagesCount         int    `json:"messages_count"`
		MsgEditWindowMins     int    `json:"msg_edit_window_mins"`
		Name                  string `json:"name"`
		OverIntegrationsLimit bool   `json:"over_integrations_limit"`
		OverStorageLimit      bool   `json:"over_storage_limit"`
		Plan                  string `json:"plan"`
		Prefs                 struct {
			AllowCalls                      bool          `json:"allow_calls"`
			AllowMessageDeletion            bool          `json:"allow_message_deletion"`
			AllowRetentionOverride          bool          `json:"allow_retention_override"`
			AllowSharedChannelPermsOverride bool          `json:"allow_shared_channel_perms_override"`
			AuthMode                        string        `json:"auth_mode"`
			CallingAppName                  string        `json:"calling_app_name"`
			ChannelHandyRxns                interface{}   `json:"channel_handy_rxns"`
			ComplianceExportStart           int           `json:"compliance_export_start"`
			CustomStatusDefaultEmoji        string        `json:"custom_status_default_emoji"`
			CustomStatusPresets             [][]string    `json:"custom_status_presets"`
			DefaultChannels                 []string      `json:"default_channels"`
			DefaultRxns                     []string      `json:"default_rxns"`
			DisableFileDeleting             bool          `json:"disable_file_deleting"`
			DisableFileEditing              bool          `json:"disable_file_editing"`
			DisableFileUploads              string        `json:"disable_file_uploads"`
			DisallowPublicFileUrls          bool          `json:"disallow_public_file_urls"`
			Discoverable                    string        `json:"discoverable"`
			DisplayEmailAddresses           bool          `json:"display_email_addresses"`
			DisplayRealNames                bool          `json:"display_real_names"`
			DmRetentionDuration             int           `json:"dm_retention_duration"`
			DmRetentionType                 int           `json:"dm_retention_type"`
			DndEnabled                      bool          `json:"dnd_enabled"`
			DndEndHour                      string        `json:"dnd_end_hour"`
			DndStartHour                    string        `json:"dnd_start_hour"`
			EnterpriseDefaultChannels       []interface{} `json:"enterprise_default_channels"`
			EnterpriseMandatoryChannels     []interface{} `json:"enterprise_mandatory_channels"`
			EnterpriseMdmDateEnabled        int           `json:"enterprise_mdm_date_enabled"`
			EnterpriseMdmLevel              int           `json:"enterprise_mdm_level"`
			EnterpriseTeamCreationRequest   struct {
				IsEnabled bool `json:"is_enabled"`
			} `json:"enterprise_team_creation_request"`
			FileRetentionDuration    int    `json:"file_retention_duration"`
			FileRetentionType        int    `json:"file_retention_type"`
			GdriveEnabledTeam        bool   `json:"gdrive_enabled_team"`
			GroupRetentionDuration   int    `json:"group_retention_duration"`
			GroupRetentionType       int    `json:"group_retention_type"`
			HideReferers             bool   `json:"hide_referers"`
			InvitesLimit             bool   `json:"invites_limit"`
			InvitesOnlyAdmins        bool   `json:"invites_only_admins"`
			LimitReachedTs           int    `json:"limit_reached_ts"`
			Locale                   string `json:"locale"`
			LoudChannelMentionsLimit int    `json:"loud_channel_mentions_limit"`
			MsgEditWindowMins        int    `json:"msg_edit_window_mins"`
			RequireAtForMention      bool   `json:"require_at_for_mention"`
			RetentionDuration        int    `json:"retention_duration"`
			RetentionType            int    `json:"retention_type"`
			ShowJoinLeave            bool   `json:"show_join_leave"`
			TeamHandyRxns            struct {
				List []struct {
					Name  string `json:"name"`
					Title string `json:"title"`
				} `json:"list"`
				Restrict bool `json:"restrict"`
			} `json:"team_handy_rxns"`
			UsesCustomizedCustomStatusPresets bool   `json:"uses_customized_custom_status_presets"`
			WarnBeforeAtChannel               string `json:"warn_before_at_channel"`
			WhoCanArchiveChannels             string `json:"who_can_archive_channels"`
			WhoCanAtChannel                   string `json:"who_can_at_channel"`
			WhoCanAtEveryone                  string `json:"who_can_at_everyone"`
			WhoCanChangeTeamProfile           string `json:"who_can_change_team_profile"`
			WhoCanCreateChannels              string `json:"who_can_create_channels"`
			WhoCanCreateDeleteUserGroups      string `json:"who_can_create_delete_user_groups"`
			WhoCanCreateGroups                string `json:"who_can_create_groups"`
			WhoCanCreateSharedChannels        string `json:"who_can_create_shared_channels"`
			WhoCanEditUserGroups              string `json:"who_can_edit_user_groups"`
			WhoCanKickChannels                string `json:"who_can_kick_channels"`
			WhoCanKickGroups                  string `json:"who_can_kick_groups"`
			WhoCanManageGuests                struct {
				Type []string `json:"type"`
			} `json:"who_can_manage_guests"`
			WhoCanManageIntegrations struct {
				Type []string `json:"type"`
			} `json:"who_can_manage_integrations"`
			WhoCanManageSharedChannels struct {
				Type []string `json:"type"`
			} `json:"who_can_manage_shared_channels"`
			WhoCanPostGeneral          string `json:"who_can_post_general"`
			WhoCanPostInSharedChannels struct {
				Type []string `json:"type"`
			} `json:"who_can_post_in_shared_channels"`
			WhoHasTeamVisibility string `json:"who_has_team_visibility"`
		} `json:"prefs"`
	} `json:"team"`
	URL   string `json:"url"`
	Users []struct {
		Deleted  bool   `json:"deleted"`
		ID       string `json:"id"`
		IsBot    bool   `json:"is_bot"`
		Name     string `json:"name"`
		Presence string `json:"presence"`
		Profile  struct {
			AvatarHash         string      `json:"avatar_hash"`
			Email              string      `json:"email"`
			Fields             interface{} `json:"fields"`
			FirstName          string      `json:"first_name"`
			Image192           string      `json:"image_192"`
			Image24            string      `json:"image_24"`
			Image32            string      `json:"image_32"`
			Image48            string      `json:"image_48"`
			Image512           string      `json:"image_512"`
			Image72            string      `json:"image_72"`
			LastName           string      `json:"last_name"`
			RealName           string      `json:"real_name"`
			RealNameNormalized string      `json:"real_name_normalized"`
		} `json:"profile"`
		TeamID  string `json:"team_id"`
		Updated int    `json:"updated"`
	} `json:"users"`
}
