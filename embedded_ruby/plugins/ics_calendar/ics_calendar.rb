require 'icalendar/recurrence'

# Define Calendar::Helpers module first (dependency for Calendar::Ics)
module Calendar
  module Helpers
    # rubocop:disable Lint/DuplicateBranch
    def cutoff_time
      case [event_layout, include_past_event?]
      when ['default', true], ['today_only', true], ['week', false], ['month', false], ['rolling_month', false]
        beginning_of_day
      when ['default', false], ['today_only', false] # this is useful for busy calendar to remove event that have elapsed.
        now_in_tz
      when ['week', true], ['month', true], ['rolling_month', true]
        time_min
      else beginning_of_day
      end
    end
    # rubocop:enable Lint/DuplicateBranch

    def event_layout = settings['event_layout']

    def include_past_event? = settings['include_past_events'] == 'yes'

    def now_in_tz = user.datetime_now

    # required to ensure locals data has a 'diff' at least 1x per day
    # given week/month view 'highlight' current day
    # without this local, previous day will be highlighted if events dont change
    def today_in_tz = now_in_tz.to_date.to_s

    def beginning_of_day = now_in_tz.beginning_of_day

    def time_min
      days_behind = case event_layout
                    when 'month', 'rolling_month'
                      30 # TODO: improve this to not store all 30 days, as right now we endup storing 30+3=60 days events.
                    else
                      7
                    end

      (beginning_of_day - days_behind.days)
    end

    def first_day = Date.strptime(settings['first_day'], '%a').wday

    def fixed_week = settings['fixed_week'] == 'yes'

    def ignore_based_on_status?(event)
      if event.instance_of?(Icalendar::Event)
        return true if event.status == 'CANCELLED' && event.dtstart.nil?
        return true if event.exdate.flatten.filter { |exception_date| event.dtstart&.to_datetime == exception_date.to_datetime }.present?
      end
      return false if event.status&.downcase == 'confirmed' # always include confirmed events

      # include non-confirmed events if user prefers to see them (received requests for both options)
      # if this branch is reached, event.status == [nil, 'rejected'] etc
      settings['event_status_filter'] == 'confirmed_only'
    end

    def include_description
      return true unless settings['include_description'] # backward compatible default value

      settings['include_description'] == 'yes'
    end

    def include_event_time
      return false unless settings['include_event_time'] # backward compatible default value

      settings['include_event_time'] == 'yes'
    end

    def ignored_phrases
      return [] unless settings['ignore_phrases']

      settings['ignore_phrases'].gsub("\n", "").gsub("\r", "").split(',').map(&:squish)
    end

    def time_format = settings['time_format'] || 'am/pm'

    def formatted_time
      return "%-I:%M %p" if time_format == 'am/pm'

      "%R"
    end

    def date_format
      return '%A, %B %-d' unless settings['date_format'].present? # => Monday, June 16

      # special i18n key that must be symbolized to be accepted as an arg to l(.. format:)
      return :short if settings['date_format'] == 'short'

      settings['date_format']
    end

    # ability to hard-code each day's first time slot in event_layout=week mode
    # by default we lookup the earliest event within the period, but some users
    # prefer to sacrifice morning visibility to see more throughout the day
    def scroll_time
      return settings['scroll_time'] if settings['scroll_time'].present?

      events.reject { |e| e[:all_day] }.map { |e| e[:start_full].to_time.strftime("%H:00:00") }.min || '08:00:00'
    end

    def scroll_time_end
      return settings['scroll_time_end'] if settings['scroll_time_end'].present?

      events.reject { |e| e[:all_day] }.map { |e| e[:end_full].to_time.strftime("%H:00:00") }.max || '24:00:00' # same default: https://fullcalendar.io/docs/slotMaxTime
    end

    def zoom_mode
      settings['zoom_mode'] == 'yes'
    end

    # Override the no_screen_padding? method from Formatter to handle calendar-specific logic
    def no_screen_padding?
      # Check if event_layout requires no padding
      if %w[week month rolling_month].include?(event_layout)
        return true
      end

      # Fall back to default behavior from Formatter
      super
    end

    def formulate_and_group_events_by_day(events, today_in_tz, days_to_show)
      # Parse today's date
      today = Date.parse(today_in_tz)

      # Generate the date range for the number of days to show
      date_range = (0...days_to_show).map { |i| today + i.days }

      # Group events by their date
      grouped_events = {}

      date_range.each do |date|
        # Format the date using the existing date_format helper
        formatted_date = if date_format == :short
                          date.strftime('%a %b %-d')
                        else
                          date.strftime(date_format)
                        end

        # Find events for this date
        day_events = events.select do |event|
          event_date = case event[:date_time]
                      when Date
                        event[:date_time]
                      when String
                        Date.parse(event[:date_time])
                      when Time, DateTime
                        event[:date_time].to_date
                      else
                        next
                      end
          event_date == date
        end

        grouped_events[formatted_date] = day_events
      end

      grouped_events
    end
  end
end


# module below is included in all ICS calendars (ex: Apple, Outlook, Fastmail, Nextcloud, etc)
module Calendar
  module Ics
    include Calendar::Helpers
    include IcsRruleHelper
    include IcsEventHelper

    def events
      @prepare_events ||= prepare_events
    end

    def prepare_events
      final_events = unique_events.sort_by { |e| e[:date_time] }
      Rails.logger.info "=== ICS Calendar Debug: Final Results ==="
      Rails.logger.info "Final event count: #{final_events.count}"
      if final_events.any?
        Rails.logger.info "Sample events:"
        final_events.first(3).each { |e| Rails.logger.info "  - #{e[:summary]} at #{e[:date_time]}" }
      end
      final_events
    rescue StandardError => e
      Rails.logger.error "=== ICS Calendar Debug: StandardError in prepare_events ==="
      Rails.logger.error "Error class: #{e.class}"
      Rails.logger.error "Error message: #{e.message}"
      Rails.logger.error "Error backtrace: #{e.backtrace.first(5).join("\n")}"
      handle_erroring_state("ics_url is invalid")
      []
    rescue ArgumentError => e
      handle_erroring_state(e.message) if e.message.include?("DNS name: nil")
      []
    rescue Icalendar::Parser::ParseError => e
      handle_erroring_state(e.message)
      []
    end

    def all_events
      @all_events ||= begin
        recurring_overrides = fetch_recurring_overrides
        all_evts = []
        recurring_count = 0
        regular_count = 0

        Rails.logger.info "=== ICS Calendar Debug: Processing events ==="
        Rails.logger.info "Recurring override count: #{recurring_overrides.count}"

        calendars.each do |cal|
          Rails.logger.info "Processing calendar with #{cal.events.count} events"
          cal.events.each do |event|
            next unless event
            next if event.respond_to?(:recurrence_id) && event.recurrence_id

            if event.rrule.present?
              Rails.logger.info "Found recurring event: #{event.summary} with RRULE: #{event.rrule.first}"
              expanded_events = occurrences(event)
              Rails.logger.info "Expanded to #{expanded_events.count} occurrences"
              expanded_events.each do |recurring_event|
                # Handle both Event and Occurrence objects for the cache key
                event_start_time = if recurring_event.is_a?(Icalendar::Recurrence::Occurrence)
                  recurring_event.start_time
                else
                  recurring_event.dtstart
                end

                key = "#{event.uid}-#{event_start_time.in_time_zone(time_zone)}"
                prepared = prepare_event(recurring_overrides[key] || recurring_event)
                if prepared
                  all_evts << prepared
                  recurring_count += 1
                else
                  Rails.logger.info "Recurring event filtered out: #{prepared ? prepared[:summary] : 'unknown'}"
                end
              end
            else
              # process regular upcoming events
              prepared = prepare_event(event)
              if prepared
                all_evts << prepared
                regular_count += 1
              else
                Rails.logger.info "Regular event filtered out: #{event.summary}"
              end
            end
          end
        end

        Rails.logger.info "Final event counts - Recurring: #{recurring_count}, Regular: #{regular_count}, Total: #{all_evts.compact.count}"
        all_evts
      end
    end

    def fetch_recurring_overrides
      overrides = {}
      calendars.each do |cal|
        cal.events.each do |event|
          next unless event

          if event.respond_to?(:recurrence_id) && event.recurrence_id
            overrides["#{event.uid}-#{event.recurrence_id.in_time_zone(time_zone)}"] = event
          end
        end
      end
      overrides
    end

    def filtered_events
      all_events_list = all_events.compact.uniq
      Rails.logger.info "=== ICS Calendar Debug: Filtering events ==="
      Rails.logger.info "Events before filtering: #{all_events_list.count}"
      Rails.logger.info "Filter date range: #{time_min} to #{time_max}"
      Rails.logger.info "Event layout: #{event_layout}, User timezone: #{time_zone}"

      filtered = all_events_list.select do |event|
        in_range = if event[:all_day]
          event[:date_time].between?(time_min, time_max)
        else
          event[:date_time].between?(time_min, time_max) || event[:end_full]&.between?(time_min, time_max)
        end

        unless in_range
          Rails.logger.info "Event filtered out by date range: #{event[:summary]} at #{event[:date_time]}"
        end

        in_range
      end

      Rails.logger.info "Events after date filtering: #{filtered.count}"
      filtered
    end

    # de-duplicates events where every param (except calname) matches -- helpful for family calendars where multiple entries otherwise exist for same event
    def unique_events
      filtered_events.compact.uniq { |evt| evt.values_at(:summary, :description, :status, :date_time, :all_day, :start_full, :end_full, :start, :end) }
    end

    def prepare_event(event)
      # Handle both Icalendar::Event and Icalendar::Recurrence::Occurrence objects
      is_occurrence = event.is_a?(Icalendar::Recurrence::Occurrence)

      # For metadata, use the event object directly (both Event and Occurrence have the needed properties)
      original_event = event

      if event_should_be_ignored?(original_event)
        Rails.logger.info "Event ignored: #{original_event.summary} (should be ignored)"
        return
      end

      # Get start time - different methods for Event vs Occurrence
      start_time = if is_occurrence
        event.start_time
      else
        event.dtstart
      end

      unless start_time
        Rails.logger.info "Event ignored: #{original_event.summary} (no start time)"
        return
      end

      # Get end time - different methods for Event vs Occurrence
      end_time = if is_occurrence
        event.end_time
      else
        event.dtend
      end

      # some params below are only needed for 1 or more event_layout options but not all
      # however all must be included as user may set event_layout==week, then create a mashup with event_layout==default
      layout_params = {
        start_full: start_time&.in_time_zone(time_zone),
        end_full: end_time&.in_time_zone(time_zone) || (start_time&.in_time_zone(time_zone) + 1.hour),
        start: start_time&.in_time_zone(time_zone)&.strftime(formatted_time),
        end: end_time&.in_time_zone(time_zone)&.strftime(formatted_time)
      }

      prepared_event = {
        summary: original_event.summary.to_s || 'Busy', # likely a private event that doesn't share full details with connected calendar
        description: sanitize_description(original_event.description),
        status: original_event.status.to_s,
        date_time: start_time.in_time_zone(time_zone),
        all_day: is_occurrence ? false : all_day_event?(original_event), # Occurrences are typically not all-day
        calname: calname(original_event)
      }.merge(layout_params)

      Rails.logger.info "Prepared event: #{prepared_event[:summary]} at #{prepared_event[:date_time]} (#{is_occurrence ? 'occurrence' : 'original'})"
      prepared_event
    end

    def recurring_event_start_date
      case event_layout
      when 'default', 'today_only', 'schedule'
        today_in_tz.to_date
      when 'week'
        today_in_tz.to_date - 7.days
      when 'month'
        today_in_tz.to_date.beginning_of_month
      when 'rolling_month'
        today_in_tz.to_date.beginning_of_week # today could be wednesday, but 'first_day' could be monday, so need earlier events
      end
    end

    def recurring_event_end_date
      case event_layout
      when 'today_only'
        today_in_tz.to_date + 2.days
      when 'default', 'week', 'month', 'rolling_month', 'schedule'
        time_max.to_date
      end
    end

    def calendars
      @calendars ||= begin
        cal_urls = line_separated_string_to_array(settings['ics_url']).map { it.gsub('webcal', 'https') }
        Rails.logger.info "=== ICS Calendar Debug: Fetching #{cal_urls.count} calendar URLs ==="
        cals = []
        cal_urls.each do |url|
          Rails.logger.info "Fetching calendar from: #{url}"
          response = fetch(url, headers:, timeout: 30, should_retry: false)
          next if response == nil # rubocop:disable Style/NilComparison
          next if response.body.nil? || response.body.empty?

          parsed_cal = Icalendar::Calendar.parse(response&.body&.gsub('Customized Time Zone', time_zone)).first
          Rails.logger.info "Parsed calendar with #{parsed_cal&.events&.count || 0} events"
          cals << parsed_cal
        end

        raise StandardError, "No calendars found" if cals.compact.empty?

        Rails.logger.info "Total calendars loaded: #{cals.compact.count}"
        cals.uniq.compact
      end
    end

    def time_zone = user.tz || 'America/New_York'

    def time_max
      days_ahead = case event_layout
                   when 'month', 'rolling_month'
                     30 # don't simply get remainder of month; FullCalendar 'previews' next month near the end
                   when 'schedule'
                     14
                   else
                     7
                   end

      (now_in_tz.end_of_day + days_ahead.days)
    end

    def headers
      return {} unless settings['headers']

      string_to_hash(settings['headers'])
    end
  end
end

# all ICS calendars look like this; they compile a hash of ~6 values and collections
module Plugins
  class IcsCalendar < Base
    include Calendar::Helpers
    include Calendar::Ics

    def locals
      { events:, event_layout:, include_description:, include_event_time:, first_day:, scroll_time:, scroll_time_end:, time_format:, today_in_tz: beginning_of_day, zoom_mode: }
    end
  end
end
