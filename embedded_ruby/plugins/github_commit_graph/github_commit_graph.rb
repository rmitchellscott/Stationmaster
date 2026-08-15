module Plugins
  class GithubCommitGraph < Base
  # Description: Flex your coding frequency
    def locals
      { username:, contributions:, stats: }
    end

    private

    def username = settings['username']

    def contributions
      @contributions ||= begin
        query = "query($userName:String!) {
          user(login: $userName){
            contributionsCollection {
              contributionCalendar {
                totalContributions
                weeks {
                  contributionDays {
                    contributionCount
                    date
                  }
                }
              }
            }
          }
        }"
        body = {
          query: query,
          variables: { userName: settings['username'] }
        }

        url = 'https://api.github.com/graphql'
        Rails.logger.info "Making GitHub API request to #{url}"
        Rails.logger.info "Request body: #{body.to_json}"
        Rails.logger.info "Headers: #{headers.inspect}"
        
        resp = HTTParty.post(url, body: body.to_json, headers: headers)
        Rails.logger.info "GitHub API response status: #{resp.code}"
        Rails.logger.info "GitHub API response: #{resp.inspect}"
        
        unless resp.success?
          Rails.logger.error "GitHub API request failed with status #{resp.code}: #{resp.body}"
          raise "GitHub API request failed: #{resp.code} - #{resp.message}"
        end
        
        if resp['errors']
          Rails.logger.error "GitHub API returned errors: #{resp['errors']}"
          raise "GitHub API errors: #{resp['errors'].map { |e| e['message'] }.join(', ')}"
        end
        
        data = resp.dig('data', 'user', 'contributionsCollection', 'contributionCalendar')
        
        unless data
          Rails.logger.error "No contribution calendar data found in response"
          Rails.logger.error "Response data structure: #{resp['data'].inspect}"
          raise "No contribution calendar data found for user #{settings['username']}"
        end

        {
          total: data['totalContributions'] || 0,
          commits: data['weeks'] || []
        }
      end
    end

    def headers
      { 
        'authorization' => "Bearer #{Rails.application.credentials.plugins.github_commit_graph_token}",
        'content-type' => 'application/json'
      }
    end

    def stats
      days = contributions[:commits].flat_map { |week| week['contributionDays'] }
      sorted_days = days.sort_by { |day| Date.parse(day['date']) }

      {
        longest_streak: longest_streak(sorted_days),
        current_streak: current_streak(sorted_days),
        max_contributions: days.map { |day| (day&.dig('contributionCount') || day&.dig(:contributionCount) || 0) }.max || 0,
        average_contributions: average_contributions(days)
      }
    end

    def average_contributions(days)
      return 0.0 if days.empty?
      total_contributions = days.sum { |day| (day&.dig('contributionCount') || day&.dig(:contributionCount) || 0) }
      (total_contributions.to_f / days.size).round(2)
    end

    def longest_streak(days)
      longest = current = 0
      days.each do |day|
        count = day&.dig('contributionCount') || day&.dig(:contributionCount) || 0
        if count.positive?
          current += 1
          longest = [longest, current].max
        else
          current = 0
        end
      end
      longest
    end

    def current_streak(days)
      streak = 0
      return streak if days.empty?
      
      # The current day can count towards the streak but it shouldn't break the streak
      last_day_count = days.last&.dig('contributionCount') || days.last&.dig(:contributionCount) || 0
      streak += 1 if last_day_count.positive?
      
      days[0..-2].reverse_each do |day|
        count = day&.dig('contributionCount') || day&.dig(:contributionCount) || 0
        break if count.zero?

        streak += 1
      end
      streak
    end
  end
end
