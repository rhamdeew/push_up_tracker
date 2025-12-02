document.addEventListener('DOMContentLoaded', function() {
    let todayData = null;
    let streakData = null;
    let calendarData = null;
    let showPreviousMonths = false;

    // Load initial data
    loadTodayData();
    loadStreakData();
    loadCalendarData();

    // Set up event listeners
    document.getElementById('completeBtn').addEventListener('click', completeToday);
    
    // Add toggle button event listener after calendar is loaded
    setTimeout(() => {
        const toggleBtn = document.getElementById('togglePreviousMonths');
        if (toggleBtn) {
            toggleBtn.addEventListener('click', togglePreviousMonths);
        }
    }, 100);

    async function loadTodayData() {
        try {
            const response = await fetch('/api/today');
            todayData = await response.json();
            updateTodayUI();
        } catch (error) {
            console.error('Error loading today data:', error);
        }
    }

    async function loadStreakData() {
        try {
            const response = await fetch('/api/streak');
            streakData = await response.json();
            updateStreakUI();
        } catch (error) {
            console.error('Error loading streak data:', error);
        }
    }

    async function loadCalendarData() {
        try {
            const currentYear = new Date().getFullYear();
            const response = await fetch(`/api/calendar?year=${currentYear}`);
            calendarData = await response.json();
            updateCalendarUI();
        } catch (error) {
            console.error('Error loading calendar data:', error);
        }
    }

    function updateTodayUI() {
        if (!todayData) return;

        const todayCount = document.getElementById('todayCount');
        const headerToday = document.getElementById('headerToday');
        const todayStatus = document.getElementById('todayStatus');
        const completeBtn = document.getElementById('completeBtn');

        todayCount.textContent = todayData.count;
        headerToday.textContent = todayData.count;

        if (todayData.done) {
            todayStatus.innerHTML = '<span class="status-dot"></span><span class="status-text">Completed!</span>';
            todayStatus.classList.add('completed');
            completeBtn.disabled = true;
            completeBtn.classList.add('completed');
            completeBtn.innerHTML = '<span class="btn-text">COMPLETED</span><span class="btn-icon"><svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 10L8 15L17 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/></svg></span>';
        } else {
            todayStatus.innerHTML = '<span class="status-dot"></span><span class="status-text">Not completed</span>';
            todayStatus.classList.remove('completed');
            completeBtn.disabled = false;
            completeBtn.classList.remove('completed');
            completeBtn.innerHTML = '<span class="btn-text">COMPLETE WORKOUT</span><span class="btn-icon"><svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 10L8 15L17 6" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/></svg></span>';
        }
    }

    function updateStreakUI() {
        if (!streakData) return;

        document.getElementById('currentStreak').textContent = streakData.current || 0;
        document.getElementById('longestStreak').textContent = streakData.longest || 0;
    }

    function updateCalendarUI() {
        if (!calendarData) return;

        const calendarContainer = document.getElementById('calendar');
        const calendarYear = document.getElementById('calendarYear');
        
        calendarYear.textContent = calendarData.year;

        // Create toggle button
        const toggleHTML = '<div class="calendar-controls">' +
            '<button id="togglePreviousMonths" class="toggle-btn">' +
            'SHOW PREVIOUS MONTHS' +
            '</button>' +
            '</div>';

        // Create calendar grid
        const calendarHTML = createCalendarGrid(calendarData);
        calendarContainer.innerHTML = toggleHTML + calendarHTML;
        
        // Add toggle button event listener
        document.getElementById('togglePreviousMonths').addEventListener('click', togglePreviousMonths);
    }

    function togglePreviousMonths() {
        showPreviousMonths = !showPreviousMonths;
        const previousMonths = document.querySelectorAll('.calendar-month.previous-month');
        const toggleBtn = document.getElementById('togglePreviousMonths');
        
        previousMonths.forEach(month => {
            if (showPreviousMonths) {
                month.style.display = 'block';
            } else {
                month.style.display = 'none';
            }
        });
        
        if (showPreviousMonths) {
            toggleBtn.textContent = 'HIDE PREVIOUS MONTHS';
        } else {
            toggleBtn.textContent = 'SHOW PREVIOUS MONTHS';
        }
    }

    function createCalendarGrid(data) {
        const currentYear = data.year;
        const months = ['January', 'February', 'March', 'April', 'May', 'June', 
                       'July', 'August', 'September', 'October', 'November', 'December'];
        const dayHeaders = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
        
        let html = '<div class="calendar-grid-container">';
        const today = new Date();
        // Get today's date string in local timezone to match backend
        const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
        const currentMonth = today.getMonth();
        
        // Start from the first record month
        const startMonth = data.startMonth || 0;
        const startYear = data.startYear || currentYear;
        
        // If we're viewing a different year, start from January
        let firstMonthToShow = (startYear === currentYear) ? startMonth : 0;
        
        for (let month = firstMonthToShow; month < 12; month++) {
            const firstDay = new Date(currentYear, month, 1);
            const lastDay = new Date(currentYear, month + 1, 0);
            
            // Check if this is a previous month (before current month AND before start month)
            const isPreviousMonth = month < currentMonth && month >= startMonth;
            const monthClass = isPreviousMonth ? 'calendar-month previous-month' : 'calendar-month';
            
            html += `<div class="${monthClass}">`;
            html += `<div class="month-header">${months[month]}</div>`;
            
            // Day headers
            html += '<div class="calendar-header">';
            for (const day of dayHeaders) {
                html += `<div class="calendar-header-cell">${day}</div>`;
            }
            html += '</div>';
            
            html += '<div class="calendar-grid">';
            
            // Empty cells for days before month starts
            for (let i = 0; i < firstDay.getDay(); i++) {
                html += '<div class="calendar-day empty"></div>';
            }
            
            // Days of the month
            for (let day = 1; day <= lastDay.getDate(); day++) {
                const date = new Date(currentYear, month, day);
                // Create date string in local timezone to match backend format
                const dateStr = `${currentYear}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
                const isToday = dateStr === todayStr;
                
                const dayData = data.days[dateStr];
                const isCompleted = dayData && dayData.done;
                
                let classes = ['calendar-day'];
                if (isToday) classes.push('today');
                if (isCompleted) classes.push('completed');
                
                html += `<div class="${classes.join(' ')}">`;
                html += `<span class="day-number">${day}</span>`;
                html += '</div>';
            }
            
            html += '</div>'; // calendar-grid
            html += '</div>'; // calendar-month
        }
        
        html += '</div>'; // calendar-grid-container
        return html;
    }

    async function completeToday() {
        if (!todayData || todayData.done) return;

        try {
            const response = await fetch('/api/today/complete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            if (response.ok) {
                todayData = await response.json();
                updateTodayUI();
                // Reload streak data as it may have changed
                loadStreakData();
                // Reload calendar to show today as completed
                loadCalendarData();
            } else {
                console.error('Error completing today\'s push-ups');
            }
        } catch (error) {
            console.error('Error completing today\'s push-ups:', error);
        }
    }
});